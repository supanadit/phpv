package args

import (
	"reflect"
	"testing"

	"github.com/spf13/cobra"
	"go.uber.org/fx"
)

// testLifecycle is a no-op fx.Lifecycle; the test never starts the app.
type testLifecycle struct{}

func (*testLifecycle) Append(fx.Hook) {}

// handlerFieldFor maps a HandlerDeps field to the PHPHandler field it must be
// forwarded to. New deps must be added here so the forwarding test covers them.
var handlerFieldFor = map[string]string{
	"Silo":        "siloSvc",
	"Assembler":   "assemblerSvc",
	"Registry":    "registrySvc",
	"Bundle":      "bundleSvc",
	"System":      "systemSvc",
	"Shim":        "shimSvc",
	"PECL":        "peclSvc",
	"Config":      "configSvc",
	"Doctor":      "doctorSvc",
	"Update":      "updateSvc",
	"PatchSync":   "patchSync",
	"PatchLister": "patchLister",
	"FPM":         "fpmSvc",
	"Apache":      "apacheSvc",
	"ServerAsm":   "serverAsbl",
	"ServerMgr":   "serverMgrFn",
}

// Regression: HandlerDeps gained FPM/Apache/ServerAsm/ServerMgr, but the
// composition root and every test built PHPHandler by struct literal, so
// nothing proved NewPHPHandler forwarded them. `phpv server install apache`
// then dereferenced a nil ServerAssembler and panicked. Every service dep
// must reach the handler.
func TestNewPHPHandler_ForwardsEveryServiceDep(t *testing.T) {
	deps := HandlerDeps{RootCmd: &cobra.Command{}, Lifecycle: &testLifecycle{}}
	depsType := reflect.TypeOf(deps)
	v := reflect.ValueOf(&deps).Elem()

	for depField := range handlerFieldFor {
		f, ok := depsType.FieldByName(depField)
		if !ok {
			t.Fatalf("HandlerDeps has no field %q (stale handlerFieldFor entry)", depField)
		}
		switch f.Type.Kind() {
		case reflect.Interface:
			// struct{ Iface } gets the interface's methods promoted, so a
			// value of it is a non-nil implementation that is never invoked.
			st := reflect.StructOf([]reflect.StructField{{Name: depField, Type: f.Type, Anonymous: true}})
			v.FieldByName(depField).Set(reflect.New(st).Elem())
		case reflect.Func:
			v.FieldByName(depField).Set(reflect.MakeFunc(f.Type, func([]reflect.Value) []reflect.Value {
				outs := make([]reflect.Value, f.Type.NumOut())
				for i := range outs {
					outs[i] = reflect.Zero(f.Type.Out(i))
				}
				return outs
			}))
		default:
			t.Fatalf("HandlerDeps.%s is %s, cannot build a sentinel", depField, f.Type)
		}
	}

	h := NewPHPHandler(deps)
	hv := reflect.ValueOf(h).Elem()

	for depField, handlerField := range handlerFieldFor {
		got := hv.FieldByName(handlerField)
		if !got.IsValid() {
			t.Fatalf("PHPHandler has no field %q (mapped from HandlerDeps.%s)", handlerField, depField)
		}
		if got.IsNil() {
			t.Fatalf("HandlerDeps.%s was not forwarded: PHPHandler.%s is nil", depField, handlerField)
		}
	}

	// Guard against future drift: every service dep must be mapped.
	for i := 0; i < depsType.NumField(); i++ {
		name := depsType.Field(i).Name
		if name == "Version" || name == "RootCmd" || name == "Lifecycle" {
			continue
		}
		if _, ok := handlerFieldFor[name]; !ok {
			t.Fatalf("HandlerDeps.%s is not mapped in handlerFieldFor; wire it and add it to the test", name)
		}
	}
}
