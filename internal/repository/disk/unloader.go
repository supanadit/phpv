package disk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/supanadit/phpv/domain"
	"github.com/ulikunitz/xz"
)

var (
	ErrUnknownFormat = errors.New("unknown archive format")
	ErrExtractFailed = errors.New("failed to extract archive")
)

type UnloadRepository struct{}

func NewUnloadRepository() *UnloadRepository {
	return &UnloadRepository{}
}

func (r *UnloadRepository) Unpack(source, destination string) (*domain.Unload, error) {
	hasTrailingSlash := strings.HasSuffix(source, "/")
	format := detectFormat(source)
	if format == "" {
		return nil, ErrUnknownFormat
	}

	source = strings.TrimRight(source, "/")

	if err := os.MkdirAll(destination, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extracted int
	var err error
	stripPrefix := !hasTrailingSlash

	switch format {
	case domain.UnloadFormatTarGz:
		extracted, err = unpackTarGz(source, destination, stripPrefix)
	case domain.UnloadFormatTarXz:
		extracted, err = unpackTarXz(source, destination, stripPrefix)
	case domain.UnloadFormatZip:
		extracted, err = unpackZip(source, destination, stripPrefix)
	default:
		return nil, ErrUnknownFormat
	}

	if err != nil {
		return nil, err
	}

	return &domain.Unload{
		Source:      source,
		Destination: destination,
		Extracted:   extracted,
	}, nil
}

func detectFormat(source string) string {
	source = strings.TrimRight(source, "/")
	source = strings.ToLower(source)
	if strings.HasSuffix(source, ".tar.xz") {
		return domain.UnloadFormatTarXz
	}
	if strings.HasSuffix(source, ".tar.gz") || strings.HasSuffix(source, ".tgz") {
		return domain.UnloadFormatTarGz
	}
	if strings.HasSuffix(source, ".zip") {
		return domain.UnloadFormatZip
	}
	return ""
}

func unpackTarGz(source, destination string, stripPrefix bool) (int, error) {
	f, err := os.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	gr, err := gzip.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gr.Close()

	return extractTar(tar.NewReader(gr), destination, stripPrefix)
}

func unpackTarXz(source, destination string, stripPrefix bool) (int, error) {
	f, err := os.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}
	defer f.Close()

	xr, err := xz.NewReader(f)
	if err != nil {
		return 0, fmt.Errorf("failed to create xz reader: %w", err)
	}

	return extractTar(tar.NewReader(xr), destination, stripPrefix)
}

func unpackZip(source, destination string, stripPrefix bool) (int, error) {
	zr, err := zip.OpenReader(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer zr.Close()

	prefix := ""
	if stripPrefix {
		prefix = commonPrefix(zr.File)
	}

	extracted := 0
	for _, file := range zr.File {
		name := strings.TrimPrefix(file.Name, prefix)
		if name == "" {
			continue
		}
		if file.FileInfo().IsDir() {
			os.MkdirAll(filepath.Join(destination, name), 0o755)
			continue
		}
		if err := extractZipFile(file, name, destination); err != nil {
			return extracted, err
		}
		extracted++
	}

	return extracted, nil
}

func commonPrefix(files []*zip.File) string {
	if len(files) == 0 {
		return ""
	}
	parts := strings.SplitN(files[0].Name, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	prefix := parts[0]
	for _, f := range files[1:] {
		if !strings.HasPrefix(f.Name, prefix+"/") {
			return ""
		}
	}
	return prefix
}

func extractTar(tr *tar.Reader, destination string, stripPrefix bool) (int, error) {
	var headers []*tar.Header
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, fmt.Errorf("failed to read tar header: %w", err)
		}
		headers = append(headers, header)
	}

	if len(headers) == 0 {
		return 0, nil
	}

	prefix := ""
	if stripPrefix {
		prefix = topLevelPrefix(headers)
	}

	extracted := 0

	for _, header := range headers {
		name := strings.TrimPrefix(header.Name, prefix)
		if name == "" {
			continue
		}

		path := filepath.Join(destination, name)
		if header.FileInfo().IsDir() {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return extracted, err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return extracted, err
		}

		f, err := os.Create(path)
		if err != nil {
			return extracted, err
		}

		_, err = io.Copy(f, &headerReader{header: header, r: tr})
		f.Close()
		if err != nil {
			return extracted, err
		}
		extracted++
	}
	return extracted, nil
}

type headerReader struct {
	header *tar.Header
	r      *tar.Reader
	read   int64
}

func (hr *headerReader) Read(p []byte) (n int, err error) {
	if hr.read >= hr.header.Size {
		return 0, io.EOF
	}
	remaining := hr.header.Size - hr.read
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}
	n, err = hr.r.Read(p)
	hr.read += int64(n)
	return n, err
}

func topLevelPrefix(headers []*tar.Header) string {
	if len(headers) == 0 {
		return ""
	}
	parts := strings.SplitN(headers[0].Name, "/", 2)
	if len(parts) < 2 {
		return ""
	}
	prefix := parts[0]
	for _, h := range headers[1:] {
		if !strings.HasPrefix(h.Name, prefix+"/") {
			return ""
		}
	}
	return prefix
}

func extractZipFile(file *zip.File, name, destination string) error {
	path := filepath.Join(destination, name)

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if file.FileInfo().IsDir() {
		return os.MkdirAll(path, 0o755)
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, rc)
	return err
}
