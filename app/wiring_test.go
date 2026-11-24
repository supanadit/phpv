package main

import (
	"reflect"
	"testing"

	"github.com/supanadit/phpv/internal/terminal/args"
)

// Regression: HandlerDeps gained FPM/Apache/ServerAsm/ServerMgr while the fx
// graph provided them, but terminalDeps never requested them, so wireTerminal
// could only forward nil and `phpv server install apache` panicked. Every
// service the handler needs must be requested from the fx graph.
func TestTerminalDeps_RequestEveryHandlerService(t *testing.T) {
	handlerType := reflect.TypeOf(args.HandlerDeps{})
	depsType := reflect.TypeOf(terminalDeps{})
	for i := 0; i < handlerType.NumField(); i++ {
		name := handlerType.Field(i).Name
		if name == "Version" || name == "RootCmd" || name == "Lifecycle" {
			continue
		}
		if _, ok := depsType.FieldByName(name); !ok {
			t.Fatalf("terminalDeps has no field %q: HandlerDeps.%s will be nil at runtime; add it to terminalDeps and forward it in wireTerminal", name, name)
		}
	}
}
