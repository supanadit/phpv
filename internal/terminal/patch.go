package terminal

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/supanadit/phpv/internal/repository"
	"github.com/supanadit/phpv/internal/repository/patches"
	"github.com/supanadit/phpv/patchsync"
)

// parsePatchScope parses the shared [package] [version] positional form:
// two arguments are package and version; a single bare argument is a
// version name scoped to the php package (e.g. `phpv patch update 7.4`).
func parsePatchScope(args []string) (*patchsync.Scope, error) {
	switch len(args) {
	case 0:
		return nil, nil
	case 1:
		return &patchsync.Scope{Package: "php", Version: args[0]}, nil
	case 2:
		return &patchsync.Scope{Package: args[0], Version: args[1]}, nil
	default:
		return nil, fmt.Errorf("expected at most [package] [version], got %v", args)
	}
}

func (h *PHPHandler) patchCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "patch",
		Short: "Manage the runtime patch tree",
		Long: "Patches are runtime data synced from the phpv repository's main branch into\n" +
			"$PHPV_ROOT/patches/; user-authored overrides live in $PHPV_ROOT/patches-local/.",
	}

	update := &cobra.Command{
		Use:     "update [package] [version]",
		Aliases: []string{"fetch"},
		Short:   "Sync the managed patch tree (delta download from main)",
		Long: "Refreshes $PHPV_ROOT/patches/ from the publication source. Only files that\n" +
			"are new or changed relative to the local manifest are downloaded, and remote\n" +
			"deletions are applied. An optional scope limits the delta: a bare argument is\n" +
			"a PHP version (e.g. `phpv patch update 7.4`), or give [package] [version].",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := parsePatchScope(args)
			if err != nil {
				return err
			}
			res, err := h.patchSync.Update(h.ctx, scope)
			if err != nil {
				return err
			}
			if res.UpToDate {
				fmt.Println("Patch tree is already up to date.")
				return nil
			}
			if scope != nil && len(res.Downloaded) == 0 && len(res.Deleted) == 0 {
				fmt.Printf("No patches match the requested scope (%s).\n", scopeLabel(*scope))
				return nil
			}
			for _, p := range res.Downloaded {
				fmt.Printf("Downloaded %s\n", p)
			}
			for _, p := range res.Deleted {
				fmt.Printf("Removed %s\n", p)
			}
			fmt.Printf("Patch tree synced to revision %s.\n", shortRev(res.Revision))
			return nil
		},
	}

	status := &cobra.Command{
		Use:   "status",
		Short: "Show patch tree revision, check time, and update availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			st, err := h.patchSync.Status(h.ctx)
			if err != nil {
				// Report the recorded local state even when the remote
				// cannot be reached.
				fmt.Printf("Recorded revision: %s\n", orNone(st.Revision))
				fmt.Printf("Files synced:      %d\n", st.Files)
				if !st.LastCheck.IsZero() {
					fmt.Printf("Last check:        %s\n", st.LastCheck.Format("2006-01-02 15:04:05"))
				}
				return err
			}
			fmt.Printf("Recorded revision: %s\n", orNone(st.Revision))
			fmt.Printf("Remote revision:   %s\n", orNone(st.RemoteRev))
			fmt.Printf("Files synced:      %d\n", st.Files)
			if st.LastCheck.IsZero() {
				fmt.Println("Last check:        never")
			} else {
				fmt.Printf("Last check:        %s", st.LastCheck.Format("2006-01-02 15:04:05"))
				if st.InLease {
					fmt.Print(" (inside stale-lease window)")
				}
				fmt.Println()
			}
			if st.UpdateDue {
				fmt.Println("An update is available: run `phpv patch update`.")
			} else {
				fmt.Println("Patch tree is up to date.")
			}
			return nil
		},
	}

	list := &cobra.Command{
		Use:   "list [package] [version]",
		Short: "List the patches that would apply for a package version",
		RunE: func(cmd *cobra.Command, args []string) error {
			scope, err := parsePatchScope(args)
			if err != nil {
				return err
			}
			if scope == nil {
				return fmt.Errorf("usage: phpv patch list [package] [version]")
			}
			root := repository.ResolveRoot()
			repo := patches.NewRepository(
				root+"/patches", root+"/patches-local").(*patches.Repository)
			entries := repo.EntriesFor(scope.Package, scope.Version)
			if len(entries) == 0 {
				fmt.Printf("No patches match %s@%s.\n", scope.Package, scope.Version)
				return nil
			}
			fmt.Printf("Patches for %s@%s (in application order):\n", scope.Package, scope.Version)
			for _, e := range entries {
				for _, f := range e.PatchFiles {
					fmt.Printf("  [%s] %s\n", e.Source, patchBase(f))
				}
				if e.Manifest != nil {
					fmt.Printf("  [%s] %s (flags manifest)\n", e.Source, manifestRelPath(e))
				}
			}
			return nil
		},
	}

	cmd.AddCommand(update, status, list)
	return cmd
}

func shortRev(rev string) string {
	if len(rev) > 10 {
		return rev[:10]
	}
	return rev
}

// patchTreeSummary renders the local patch-tree state for `phpv doctor`:
// managed tree presence, recorded revision, and stale-lease status. It
// never touches the network.
func patchTreeSummary() string {
	root := repository.ResolveRoot()
	managed := root + "/patches"
	rev := ""
	lastCheck := time.Time{}
	if data, err := os.ReadFile(filepath.Join(managed, patchsync.RevisionFileName)); err == nil {
		rev = strings.TrimSpace(string(data))
	}
	if data, err := os.ReadFile(filepath.Join(managed, patchsync.LastCheckFileName)); err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(data))); err == nil {
			lastCheck = t
		}
	}
	if fi, err := os.Stat(managed); err == nil && fi.IsDir() {
		files := 0
		_ = filepath.WalkDir(managed, func(p string, d os.DirEntry, err error) error {
			if err == nil && !d.IsDir() && !strings.HasPrefix(d.Name(), ".") {
				files++
			}
			return nil
		})
		lease := patchsync.DefaultLease
		if v := os.Getenv("PHPV_PATCH_LEASE"); v != "" {
			if d, err := time.ParseDuration(v); err == nil {
				lease = d
			}
		}
		status := "never checked"
		if !lastCheck.IsZero() {
			if time.Since(lastCheck) < lease {
				status = "fresh (inside stale-lease)"
			} else {
				status = "stale (lease expired)"
			}
		}
		return fmt.Sprintf("Patch tree: present at %s, %d file(s), revision %s, %s",
			managed, files, orNone(rev), status)
	}
	return fmt.Sprintf("Patch tree: not synced yet (%s); run `phpv patch update`", managed)
}

func scopeLabel(sc patchsync.Scope) string {
	return sc.Package + "@" + sc.Version
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return shortRev(s)
}

func patchBase(path string) string {
	if i := strings.LastIndex(path, "/"); i >= 0 {
		return path[i+1:]
	}
	return path
}

func manifestRelPath(e patches.Entry) string {
	return e.DirPath + "/flags.yaml"
}
