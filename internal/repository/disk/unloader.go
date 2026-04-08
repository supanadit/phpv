package disk

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/spf13/afero"
	"github.com/supanadit/phpv/domain"
	"github.com/ulikunitz/xz"
)

var (
	ErrUnknownFormat = errors.New("unknown archive format")
)

type UnloadRepository struct {
	fs afero.Fs
}

func NewUnloadRepository() *UnloadRepository {
	return &UnloadRepository{
		fs: afero.NewOsFs(),
	}
}

func NewUnloadRepositoryWithFs(fs afero.Fs) *UnloadRepository {
	return &UnloadRepository{
		fs: fs,
	}
}

func (r *UnloadRepository) ensureFs() {
	if r.fs == nil {
		r.fs = afero.NewOsFs()
	}
}

func (r *UnloadRepository) Unpack(source, destination string) (*domain.Unload, error) {
	r.ensureFs()
	hasTrailingSlash := strings.HasSuffix(source, "/")
	format := detectFormat(source)
	if format == "" {
		return nil, ErrUnknownFormat
	}

	source = strings.TrimRight(source, "/")

	if err := r.fs.MkdirAll(destination, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create destination directory: %w", err)
	}

	var extracted int
	var err error
	stripPrefix := !hasTrailingSlash

	switch format {
	case domain.UnloadFormatTarGz:
		extracted, err = r.unpackTarGz(source, destination, stripPrefix)
	case domain.UnloadFormatTarXz:
		extracted, err = r.unpackTarXz(source, destination, stripPrefix)
	case domain.UnloadFormatZip:
		extracted, err = r.unpackZip(source, destination, stripPrefix)
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

func (r *UnloadRepository) unpackTarGz(source, destination string, stripPrefix bool) (int, error) {
	f, err := r.fs.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}

	gr, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return 0, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer func() {
		gr.Close()
		f.Close()
	}()

	return r.extractTar(tar.NewReader(gr), destination, stripPrefix)
}

func (r *UnloadRepository) unpackTarXz(source, destination string, stripPrefix bool) (int, error) {
	f, err := r.fs.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open archive: %w", err)
	}

	xr, err := xz.NewReader(f)
	if err != nil {
		f.Close()
		return 0, fmt.Errorf("failed to create xz reader: %w", err)
	}
	defer f.Close()

	return r.extractTar(tar.NewReader(xr), destination, stripPrefix)
}

func (r *UnloadRepository) unpackZip(source, destination string, stripPrefix bool) (int, error) {
	f, err := r.fs.Open(source)
	if err != nil {
		return 0, fmt.Errorf("failed to open zip archive: %w", err)
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to stat zip archive: %w", err)
	}

	zr, err := zip.NewReader(f, stat.Size())
	if err != nil {
		return 0, fmt.Errorf("failed to open zip archive: %w", err)
	}

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
			r.fs.MkdirAll(filepath.Join(destination, name), 0o755)
			continue
		}
		if err := r.extractZipFile(file, name, destination); err != nil {
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

func (r *UnloadRepository) extractTar(tr *tar.Reader, destination string, stripPrefix bool) (int, error) {
	extracted := 0
	prefix := ""
	firstEntry := true

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return extracted, fmt.Errorf("failed to read tar header: %w", err)
		}

		name := header.Name
		if firstEntry && stripPrefix {
			parts := strings.SplitN(name, "/", 2)
			if len(parts) >= 2 {
				prefix = parts[0]
			}
			firstEntry = false
		}
		name = strings.TrimPrefix(name, prefix)
		if name == "" {
			io.CopyN(io.Discard, tr, header.Size)
			continue
		}

		path := filepath.Join(destination, name)
		if header.FileInfo().IsDir() {
			if err := r.fs.MkdirAll(path, 0o755); err != nil {
				return extracted, err
			}
			io.CopyN(io.Discard, tr, header.Size)
			continue
		}

		if err := r.fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return extracted, err
		}

		f, err := r.fs.Create(path)
		if err != nil {
			return extracted, err
		}

		_, err = io.CopyN(f, tr, header.Size)
		f.Close()
		if err != nil {
			return extracted, err
		}
		extracted++
	}
	return extracted, nil
}

func (r *UnloadRepository) extractZipFile(file *zip.File, name, destination string) error {
	path := filepath.Join(destination, name)

	rc, err := file.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	if file.FileInfo().IsDir() {
		return r.fs.MkdirAll(path, 0o755)
	}

	if err := r.fs.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	f, err := r.fs.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = io.Copy(f, rc)
	return err
}
