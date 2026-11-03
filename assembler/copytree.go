package assembler

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// copyTree recursively copies src to dst, preserving file modes and
// modification times. Symlinks are recreated. Used to clone the patched php
// source tree for the FPM variant build without disturbing the CLI build's
// configure state.
//
// Preserving mtimes is load-bearing: make triggers bison regeneration when a
// shipped .y parser looks newer than its pre-generated .c (fresh mtimes in
// directory-iteration order do exactly that), which breaks PHP < 7.1 builds
// against bison 3. With original timestamps the shipped generated parsers
// stay up-to-date and bison is never needed.
func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	switch info.Mode() & os.ModeType {
	case os.ModeSymlink:
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		return os.Symlink(target, dst)
	case os.ModeDir:
		if err := os.MkdirAll(dst, info.Mode().Perm()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyTree(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return os.Chtimes(dst, mtime(info), mtime(info))
	default:
		in, err := os.Open(src)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, in); err != nil {
			out.Close()
			return fmt.Errorf("copy %s: %w", filepath.Base(src), err)
		}
		if err := out.Close(); err != nil {
			return err
		}
		return os.Chtimes(dst, mtime(info), mtime(info))
	}
}

func mtime(info os.FileInfo) time.Time {
	return info.ModTime()
}
