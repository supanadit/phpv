package terminal

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/doctor"
)

func (h *PHPHandler) doctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose phpv installation issues",
		Long:  "Run diagnostic checks on the phpv installation and report any issues.",
		Args:  cobra.NoArgs,
		RunE:  h.doctor,
	}
	cmd.Flags().Bool("json", false, "Output in JSON format")
	return cmd
}

func (h *PHPHandler) doctor(cmd *cobra.Command, args []string) error {
	jsonFlag, _ := cmd.Flags().GetBool("json")

	root := h.siloSvc.GetSilo().Root
	issues := doctor.Check(root)

	if jsonFlag {
		type doctorResponse struct {
			Issues []doctor.Issue `json:"issues"`
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: doctorResponse{Issues: issues}})
	}

	if len(issues) == 0 {
		fmt.Println("✓ No issues found")
		return nil
	}

	var criticalCount int
	for _, s := range []doctor.Severity{doctor.SeverityCritical, doctor.SeverityWarning, doctor.SeverityInfo} {
		for _, issue := range issues {
			if issue.Severity != s {
				continue
			}
			if s == doctor.SeverityCritical {
				criticalCount++
			}
			glyph := "ℹ"
			switch issue.Severity {
			case doctor.SeverityCritical:
				glyph = "✗"
			case doctor.SeverityWarning:
				glyph = "⚠"
			case doctor.SeverityInfo:
				glyph = "ℹ"
			}
			fmt.Printf("  %s [%s] %s\n", glyph, issue.Severity, issue.Title)
			fmt.Printf("    %s\n", issue.Detail)
			if issue.Fix != "" {
				fmt.Printf("    Fix: %s\n", issue.Fix)
			}
			fmt.Println()
		}
	}

	if criticalCount > 0 {
		os.Exit(1)
	}
	return nil
}
