package apache

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/supanadit/phpv/domain"
)

// fpmConf returns the FPM pool config path for the configured server.
func fpmConf() string {
	return filepath.Join(fpmDir(), "www.conf")
}

// writeFPMPool generates a php-fpm pool config listening on 127.0.0.1:<port>.
func (s *ApacheService) writeFPMPool(cfg *domain.WebserverConfig) error {
	port := cfg.FPMBasePort
	if port == 0 {
		port = 9000
	}
	pool := fmt.Sprintf(`; phpv: PHP-FPM pool for %s
[www]
user = %s
group = %s
listen = 127.0.0.1:%d
listen.allowed_clients = 127.0.0.1
pm = dynamic
pm.max_children = 5
pm.start_servers = 2
pm.min_spare_servers = 1
pm.max_spare_servers = 3
php_admin_flag[log_errors] = on
php_admin_value[error_log] = %s
php_admin_value[upload_tmp_dir] = %s
php_admin_value[session.save_path] = %s
`, cfg.PHPVersion, cfg.User, cfg.Group, port, filepath.Join(logsDir(), "php-fpm.log"),
		filepath.Join(apacheRoot(), "tmp"), filepath.Join(apacheRoot(), "tmp"))

	if err := ensureDirs(); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(apacheRoot(), "tmp"), 0o755); err != nil {
		return err
	}
	return os.WriteFile(fpmConf(), []byte(pool), 0o644)
}

// applyServerIdentity pins the httpd.conf User/Group (handled by writeBaseConfig,
// kept as a no-op hook for symmetry with the other config steps).
func (s *ApacheService) applyServerIdentity(_ *domain.WebserverConfig, _ string) error {
	return nil
}
