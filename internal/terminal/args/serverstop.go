package args

import (
	"fmt"

	"github.com/spf13/cobra"
)

func (h *PHPHandler) serverstop() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [instance-id]",
		Short: "Stop a running server instance (or all when omitted)",
		Args:  cobra.MaximumNArgs(1),
		ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
			m, err := h.serverManager()
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			running, _ := m.Running()
			var ids []string
			for _, inst := range running {
				ids = append(ids, inst.ID)
			}
			return ids, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := h.serverManager()
			if err != nil {
				return err
			}
			if len(args) == 1 {
				inst, err := m.StopByID(args[0])
				if err != nil {
					return err
				}
				fmt.Printf("✓ stopped %s (%s:%d)\n", inst.ID, inst.Server, inst.Port)
				return nil
			}
			running, err := m.Running()
			if err != nil {
				return err
			}
			if len(running) == 0 {
				fmt.Println("no running instances")
				return nil
			}
			for _, inst := range running {
				if err := m.Stop(inst); err != nil {
					return err
				}
				fmt.Printf("✓ stopped %s (%s:%d)\n", inst.ID, inst.Server, inst.Port)
			}
			return nil
		},
	}

	return cmd
}
