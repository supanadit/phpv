package webserver

import "github.com/supanadit/phpv/domain"

// ConfigureOptions carries the options for configuring a webserver's PHP
// integration. Which fields are honored depends on the server type.
type ConfigureOptions struct {
	PHPVersion string              // PHP version to connect
	Connector  domain.ConnectorMode // fpm / cgi / mod_php
	MPM        string               // event / worker / prefork
	Port       int                  // listen port (default 80)
	User       string               // run-as user (empty = current user)
	Group      string               // run-as group (empty = current user)
}

// WebServer is the common interface implemented by each supported webserver
// (Apache is the first; nginx, caddy, etc. can implement it later).
type WebServer interface {
	Name() string
	Install(version string) error
	Uninstall() error
	IsInstalled() (bool, string)
	Configure(opts ConfigureOptions) error
	Start(foreground bool) error
	Stop() error
	Restart() error
	Status() (string, error)

	VhostAdd(vhost domain.Vhost) error
	VhostRemove(serverName string) error
	VhostList() ([]domain.Vhost, error)
	VhostEnable(serverName string) error
	VhostDisable(serverName string) error
}
