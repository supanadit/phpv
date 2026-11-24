package args

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/supanadit/phpv/domain"
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
	issues := h.doctorSvc.Check(root)

	if jsonFlag {
		type doctorResponse struct {
			Issues []domain.Issue `json:"issues"`
		}
		return printJSON(jsonResponse{SchemaVersion: 1, Data: doctorResponse{Issues: issues}})
	}

	fmt.Printf("  %s\n", patchTreeSummary())

	if len(issues) == 0 {
		fmt.Println("✓ No issues found")
		return nil
	}

	var criticalCount int
	for _, s := range []domain.Severity{domain.SeverityCritical, domain.SeverityWarning, domain.SeverityInfo} {
		for _, issue := range issues {
			if issue.Severity != s {
				continue
			}
			if s == domain.SeverityCritical {
				criticalCount++
			}
			glyph := "ℹ"
			switch issue.Severity {
			case domain.SeverityCritical:
				glyph = "✗"
			case domain.SeverityWarning:
				glyph = "⚠"
			case domain.SeverityInfo:
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
