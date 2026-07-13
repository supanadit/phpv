package pecl

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type peclPackage struct {
	Name    string
	Version string
	URL     string
}

type Downloader interface {
	ResolveLatest(name string) (peclPackage, error)
	Download(pkg peclPackage, destDir string) (string, error)
}

type peclDownloader struct {
	client *http.Client
}

func newPECLDownloader() *peclDownloader {
	return &peclDownloader{client: http.DefaultClient}
}

type peclRestResponse struct {
	Version string `xml:"v"`
}

func (d *peclDownloader) ResolveLatest(name string) (peclPackage, error) {
	url := fmt.Sprintf("https://pecl.php.net/rest/r/%s/latest.xml", name)
	resp, err := d.client.Get(url)
	if err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net resolve %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return peclPackage{}, fmt.Errorf("pecl.net resolve %s: HTTP %d", name, resp.StatusCode)
	}

	var r peclRestResponse
	if err := xml.NewDecoder(resp.Body).Decode(&r); err != nil {
		return peclPackage{}, fmt.Errorf("pecl.net parse response for %s: %w", name, err)
	}
	if r.Version == "" {
		return peclPackage{}, fmt.Errorf("pecl.net: no version found for %s", name)
	}

	return peclPackage{
		Name:    name,
		Version: r.Version,
		URL:     fmt.Sprintf("https://pecl.php.net/get/%s-%s.tgz", name, r.Version),
	}, nil
}

func (d *peclDownloader) Download(pkg peclPackage, destDir string) (string, error) {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("create pecl cache dir: %w", err)
	}

	destPath := filepath.Join(destDir, pkg.Name+"-"+pkg.Version+".tgz")

	if _, err := os.Stat(destPath); err == nil {
		return destPath, nil
	}

	resp, err := d.client.Get(pkg.URL)
	if err != nil {
		return "", fmt.Errorf("download %s: %w", pkg.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download %s: HTTP %d", pkg.URL, resp.StatusCode)
	}

	f, err := os.Create(destPath)
	if err != nil {
		return "", fmt.Errorf("create %s: %w", destPath, err)
	}
	defer f.Close()

	if _, err := io.Copy(f, resp.Body); err != nil {
		os.Remove(destPath)
		return "", fmt.Errorf("write %s: %w", destPath, err)
	}

	return destPath, nil
}
