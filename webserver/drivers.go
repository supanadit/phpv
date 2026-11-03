package webserver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
)

// Driver is the per-server behavior: config generation, start and stop.
// One implementation per supported webserver; the Manager selects by name.
type Driver interface {
	// Name is the domain server name (caddy, nginx, apache).
	Name() string
	// Binary returns the server executable inside an installed prefix.
	Binary(prefix string) string
	// WriteConfig renders the server config for the instance into
	// inst.StateDir and returns the config file path.
	WriteConfig(inst domain.ServerInstance, fpmConf string) (string, error)
	// StartArgs returns the command line to start the server with the
	// rendered config (foreground; the Manager handles detach).
	StartArgs(inst domain.ServerInstance, confPath string) []string
	// Stop terminates a running instance gracefully.
	Stop(inst domain.ServerInstance) error
}

// CaddyDriver serves PHP through php_fastcgi (reverse_proxy to FastCGI).
type CaddyDriver struct{}

func (CaddyDriver) Name() string { return domain.ServerCaddy }

func (CaddyDriver) Binary(prefix string) string {
	return FindBinary(prefix, "caddy")
}

func (CaddyDriver) WriteConfig(inst domain.ServerInstance, fpmConf string) (string, error) {
	if err := os.MkdirAll(inst.StateDir, 0o755); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "{\n  admin off\n  auto_https off\n}\n\n")
	b.WriteString("{PORT} {\n")
	host := inst.Host
	if host != "" && host != "localhost" {
		// Serve on the vhost name; port binding comes from the site label.
		fmt.Fprintf(&b, "  # vhost: %s\n", host)
	}
	fmt.Fprintf(&b, "  root * %s\n", inst.Dir)
	b.WriteString("  encode gzip\n")
	fmt.Fprintf(&b, "  php_fastcgi unix/%s\n", inst.Socket)
	b.WriteString("  file_server\n")
	b.WriteString("}\n")
	conf := filepath.Join(inst.StateDir, "Caddyfile")
	rendered := strings.ReplaceAll(b.String(), "{PORT}", ":"+fmt.Sprintf("%d", inst.Port))
	if err := os.WriteFile(conf, []byte(rendered), 0o644); err != nil {
		return "", fmt.Errorf("write Caddyfile: %w", err)
	}
	return conf, nil
}

func (CaddyDriver) StartArgs(inst domain.ServerInstance, confPath string) []string {
	return []string{"run", "--config", confPath, "--adapter", "caddyfile"}
}

func (CaddyDriver) Stop(inst domain.ServerInstance) error {
	return nil // SIGTERM on the recorded pid (Manager handles it)
}

// NginxDriver serves PHP via FastCGI pass to the instance socket.
type NginxDriver struct{}

func (NginxDriver) Name() string { return domain.ServerNginx }

func (NginxDriver) Binary(prefix string) string {
	return FindBinary(prefix, "nginx")
}

func (NginxDriver) WriteConfig(inst domain.ServerInstance, fpmConf string) (string, error) {
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "logs"), 0o755); err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "tmp", "client"), 0o755); err != nil {
		return "", err
	}
	// nginx resolves bare `include` paths against its --prefix/conf dir,
	// which an isolated silo install does not populate; ship the params
	// file next to the generated config instead.
	if err := os.WriteFile(filepath.Join(inst.StateDir, "fastcgi_params"), []byte(fastcgiParams), 0o644); err != nil {
		return "", fmt.Errorf("write fastcgi_params: %w", err)
	}
	var b strings.Builder
	b.WriteString("worker_processes 1;\n")
	b.WriteString("daemon off;\n")
	b.WriteString("pid " + inst.StateDir + "/nginx.pid;\n")
	b.WriteString("error_log " + inst.StateDir + "/logs/error.log info;\n\n")
	b.WriteString("events { worker_connections 256; }\n\n")
	b.WriteString("http {\n")
	b.WriteString("  access_log " + inst.StateDir + "/logs/access.log;\n")
	b.WriteString("  client_body_temp_path " + inst.StateDir + "/tmp/client;\n")
	b.WriteString("  proxy_temp_path " + inst.StateDir + "/tmp/proxy;\n")
	b.WriteString("  fastcgi_temp_path " + inst.StateDir + "/tmp/fastcgi;\n")
	b.WriteString("  uwsgi_temp_path " + inst.StateDir + "/tmp/uwsgi;\n")
	b.WriteString("  scgi_temp_path " + inst.StateDir + "/tmp/scgi;\n\n")
	fmt.Fprintf(&b, "  server {\n")
	fmt.Fprintf(&b, "    listen %d;\n", inst.Port)
	if inst.Host != "" && inst.Host != "localhost" {
		fmt.Fprintf(&b, "    server_name %s;\n", inst.Host)
	}
	fmt.Fprintf(&b, "    root %s;\n", inst.Dir)
	b.WriteString("    index index.php index.html;\n\n")
	b.WriteString("    location / {\n      try_files $uri $uri/ /index.php?$query_string;\n    }\n\n")
	b.WriteString("    location ~ \\.php$ {\n")
	b.WriteString("      try_files $uri =404;\n")
	fmt.Fprintf(&b, "      fastcgi_pass unix:%s;\n", inst.Socket)
	b.WriteString("      fastcgi_index index.php;\n")
	b.WriteString("      include " + filepath.Join(inst.StateDir, "fastcgi_params") + ";\n")
	b.WriteString("      fastcgi_param SCRIPT_FILENAME $document_root$fastcgi_script_name;\n")
	b.WriteString("    }\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")
	conf := filepath.Join(inst.StateDir, "nginx.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write nginx.conf: %w", err)
	}
	return conf, nil
}

func (NginxDriver) StartArgs(inst domain.ServerInstance, confPath string) []string {
	return []string{"-c", confPath}
}

func (NginxDriver) Stop(inst domain.ServerInstance) error {
	return nil
}

// ApacheDriver serves PHP via mod_proxy_fcgi to the instance socket.
type ApacheDriver struct{}

func (ApacheDriver) Name() string { return domain.ServerApache }

func (ApacheDriver) Binary(prefix string) string {
	return FindBinary(prefix, "httpd")
}

func (ApacheDriver) WriteConfig(inst domain.ServerInstance, fpmConf string) (string, error) {
	if err := os.MkdirAll(filepath.Join(inst.StateDir, "logs"), 0o755); err != nil {
		return "", err
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ServerName %s\n", hostOrLocalhost(inst.Host))
	fmt.Fprintf(&b, "Listen %d\n", inst.Port)
	b.WriteString("PidFile " + inst.StateDir + "/httpd.pid\n")
	b.WriteString("ErrorLog " + inst.StateDir + "/logs/error.log\n")
	b.WriteString("CustomLog " + inst.StateDir + "/logs/access.log combined\n\n")
	// The MPM (event) is compiled into phpv's httpd build (httpd -l), so it
	// must NOT be LoadModule'd. mod_so is likewise built in and provides the
	// LoadModule directive itself.
	b.WriteString("LoadModule authz_core_module modules/mod_authz_core.so\n")
	b.WriteString("LoadModule unixd_module modules/mod_unixd.so\n")
	b.WriteString("LoadModule log_config_module modules/mod_log_config.so\n")
	b.WriteString("LoadModule mime_module modules/mod_mime.so\n")
	b.WriteString("LoadModule dir_module modules/mod_dir.so\n")
	b.WriteString("LoadModule proxy_module modules/mod_proxy.so\n")
	b.WriteString("LoadModule proxy_fcgi_module modules/mod_proxy_fcgi.so\n")
	b.WriteString("LoadModule setenvif_module modules/mod_setenvif.so\n\n")
	b.WriteString("TypesConfig conf/mime.types\n")
	b.WriteString("DirectoryIndex index.php index.html\n\n")
	fmt.Fprintf(&b, "DocumentRoot \"%s\"\n", inst.Dir)
	fmt.Fprintf(&b, "<Directory \"%s\">\n", inst.Dir)
	b.WriteString("  Options Indexes FollowSymLinks\n")
	b.WriteString("  AllowOverride All\n")
	b.WriteString("  Require all granted\n")
	b.WriteString("</Directory>\n\n")
	b.WriteString("<FilesMatch \\.php$>\n")
	fmt.Fprintf(&b, "  SetHandler \"proxy:unix:%s|fcgi://localhost\"\n", inst.Socket)
	b.WriteString("</FilesMatch>\n")
	conf := filepath.Join(inst.StateDir, "httpd.conf")
	if err := os.WriteFile(conf, []byte(b.String()), 0o644); err != nil {
		return "", fmt.Errorf("write httpd.conf: %w", err)
	}
	return conf, nil
}

func (ApacheDriver) StartArgs(inst domain.ServerInstance, confPath string) []string {
	return []string{"-f", confPath, "-D", "FOREGROUND"}
}

func (ApacheDriver) Stop(inst domain.ServerInstance) error {
	return nil
}

func hostOrLocalhost(h string) string {
	if h == "" {
		return "localhost"
	}
	return h
}

// fastcgiParams is the standard nginx FastCGI parameter set (from the
// nginx distribution's fastcgi_params file, minus the SCRIPT_FILENAME we
// set explicitly).
const fastcgiParams = `fastcgi_param  QUERY_STRING       $query_string;
fastcgi_param  REQUEST_METHOD     $request_method;
fastcgi_param  CONTENT_TYPE       $content_type;
fastcgi_param  CONTENT_LENGTH     $content_length;
fastcgi_param  REQUEST_URI        $request_uri;
fastcgi_param  DOCUMENT_URI       $document_uri;
fastcgi_param  DOCUMENT_ROOT      $document_root;
fastcgi_param  SERVER_PROTOCOL    $server_protocol;
fastcgi_param  REQUEST_SCHEME     $scheme;
fastcgi_param  HTTPS              $https if_not_empty;
fastcgi_param  GATEWAY_INTERFACE  CGI/1.1;
fastcgi_param  SERVER_SOFTWARE    nginx;
fastcgi_param  REMOTE_ADDR        $remote_addr;
fastcgi_param  REMOTE_PORT        $remote_port;
fastcgi_param  SERVER_ADDR        $server_addr;
fastcgi_param  SERVER_PORT        $server_port;
fastcgi_param  SERVER_NAME        $server_name;
fastcgi_param  REDIRECT_STATUS    200;
`
