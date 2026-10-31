package disk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/patchsync"
)

// PatchPublicationRepo is the GitHub repository publishing the patch tree.
const PatchPublicationRepo = "supanadit/phpv"

// PatchRemote implements patchsync.Remote against the publication
// repository's main branch: the tree listing comes from the Git Trees
// API at the pinned HEAD SHA (one API call per check), and file content
// downloads go through raw.githubusercontent.com at that same SHA
// (not API-rate-limited).
type PatchRemote struct {
	httpClient *http.Client
	repo       string
	branch     string
}

func NewPatchRemote() *PatchRemote {
	return &PatchRemote{httpClient: http.DefaultClient, repo: PatchPublicationRepo, branch: "main"}
}

// ListTree returns every patch-tree file at the branch's current HEAD
// revision. The listing is filtered to the patches/ prefix; sync
// metadata files are excluded (they are runtime state, not content).
func (r *PatchRemote) ListTree(ctx context.Context) ([]patchsync.RemoteFile, string, error) {
	sha, err := r.headSHA(ctx)
	if err != nil {
		return nil, "", err
	}
	url := fmt.Sprintf("https://api.github.com/repos/%s/git/trees/%s?recursive=1", r.repo, sha)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("list patch tree: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("GitHub API: unexpected status %s", resp.Status)
	}
	var body struct {
		Tree []struct {
			Path string `json:"path"`
			Type string `json:"type"`
			SHA  string `json:"sha"`
		} `json:"tree"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return nil, "", fmt.Errorf("parse tree listing: %w", err)
	}
	var out []patchsync.RemoteFile
	for _, e := range body.Tree {
		if e.Type != "blob" || !strings.HasPrefix(e.Path, "patches/") {
			continue
		}
		rel := strings.TrimPrefix(e.Path, "patches/")
		if rel == "" || strings.HasPrefix(filepath.Base(rel), ".") {
			continue
		}
		out = append(out, patchsync.RemoteFile{Path: rel, BlobSHA: e.SHA})
	}
	return out, sha, nil
}

func (r *PatchRemote) headSHA(ctx context.Context) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s", r.repo, r.branch)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("resolve patch tree revision: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API: unexpected status %s", resp.Status)
	}
	var body struct {
		Commit struct {
			SHA string `json:"sha"`
		} `json:"commit"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", fmt.Errorf("parse branch info: %w", err)
	}
	if body.Commit.SHA == "" {
		return "", fmt.Errorf("no commit SHA for branch %s", r.branch)
	}
	return body.Commit.SHA, nil
}

// FetchBlob downloads one patch-tree file at the current HEAD via
// raw.githubusercontent.com.
func (r *PatchRemote) FetchBlob(ctx context.Context, path, dest string) error {
	url := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/patches/%s", r.repo, r.branch, path)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := r.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("download %s: %w", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download %s: unexpected status %s", path, resp.Status)
	}
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, resp.Body); err != nil {
		return fmt.Errorf("download %s: %w", path, err)
	}
	return nil
}
