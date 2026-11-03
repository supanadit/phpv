package webserver

import "os/exec"

// command and lookPath are indirections so tests can stub process calls.
var (
	command  = exec.Command
	lookPath = exec.LookPath
)
