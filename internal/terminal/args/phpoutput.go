package args

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// autoDetectResolveCmd is a hidden subcommand used by shim scripts to resolve version constraints.
func (h *PHPHandler) autoDetectResolveCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "auto-detect-resolve [version]",
		Hidden: true,
		Args:   cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			constraint := ""
			if len(args) == 1 {
				constraint = args[0]
			}
			exact, err := h.assemblerSvc.ResolveVersion("php", constraint)
			if err != nil {
				return err
			}
			fmt.Println(exact)
			return nil
		},
	}
}

// printJSON marshals v to stdout as indented JSON. All JSON responses include
// a schema_version field for forward compatibility.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// jsonResponse wraps a payload with a schema version.
type jsonResponse struct {
	SchemaVersion int `json:"schema_version"`
	Data          any `json:"data"`
}
