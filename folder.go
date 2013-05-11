package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
)

////////////////////////////////////////////////////////////////////////////////

type FxWalk func(path string) error

type InputFolder interface {
	OpenFile(path string) (io.ReadCloser, error)
	Walk(fnWalk FxWalk) error
	ReadDirNames() ([]string, error)
	Close()
}

////////////////////////////////////////////////////////////////////////////////

type SystemFolder struct {
	base string
}

func OpenSystemFolder(path string) *SystemFolder {
	return &SystemFolder{base: path}
}

func (folder *SystemFolder) OpenFile(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(folder.base, path))
}

func (folder *SystemFolder) Walk(fnWalk FxWalk) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path, _ = filepath.Rel(folder.base, path)
		return fnWalk(path)
	}

	return filepath.Walk(folder.base, walk)
}

func (folder *SystemFolder) ReadDirNames() ([]string, error) {
	f, e := os.Open(folder.base)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

func (folder *SystemFolder) Close() {
}

////////////////////////////////////////////////////////////////////////////////

type ZipFolder struct {
	zrc *zip.ReadCloser
}

func OpenZipFolder(path string) (*ZipFolder, error) {
	rc, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	return &ZipFolder{zrc: rc}, nil
}

func (folder *ZipFolder) Close() {
	folder.zrc.Close()
}

func (folder *ZipFolder) OpenFile(path string) (io.ReadCloser, error) {
	for _, f := range folder.zrc.File {
		if strings.ToLower(f.Name) == path {
			return f.Open()
		}
	}
	return nil, os.ErrNotExist
}

func (folder *ZipFolder) Walk(fnWalk FxWalk) error {
	for _, f := range folder.zrc.File {
		if e := fnWalk(f.Name); e != nil {
			return e
		}
	}
	return nil
}

func (folder *ZipFolder) ReadDirNames() ([]string, error) {
	names := make([]string, len(folder.zrc.File))
	for i, f := range folder.zrc.File {
		names[i] = f.Name
	}
	return names, nil
}

////////////////////////////////////////////////////////////////////////////////

func OpenInputFolder(path string) (InputFolder, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return nil, e
	}

	if stat.IsDir() {
		return OpenSystemFolder(path), nil
	}

	return OpenZipFolder(path)
}

////////////////////////////////////////////////////////////////////////////////
