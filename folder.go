package main

import (
	"archive/zip"
	"bytes"
	"io"
	"io/ioutil"
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
}

////////////////////////////////////////////////////////////////////////////////

type SystemFolder struct {
	path string
}

func OpenSystemFolder(path string) *SystemFolder {
	return &SystemFolder{path: path}
}

func (this *SystemFolder) OpenFile(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(this.path, path))
}

func (this *SystemFolder) Walk(fnWalk FxWalk) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path, _ = filepath.Rel(this.path, path)
		return fnWalk(path)
	}

	return filepath.Walk(this.path, walk)
}

func (this *SystemFolder) ReadDirNames() ([]string, error) {
	f, e := os.Open(this.path)
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return f.Readdirnames(-1)
}

////////////////////////////////////////////////////////////////////////////////

type ZipFolder struct {
	zr *zip.Reader
}

func NewZipFolder(data []byte) (*ZipFolder, error) {
	r := bytes.NewReader(data)
	if zr, e := zip.NewReader(r, int64(len(data))); e != nil {
		return nil, e
	} else {
		return &ZipFolder{zr: zr}, nil
	}
}

func OpenZipFolder(path string) (*ZipFolder, error) {
	if data, e := ioutil.ReadFile(path); e != nil {
		return nil, e
	} else {
		return NewZipFolder(data)
	}
}

func (this *ZipFolder) OpenFile(path string) (io.ReadCloser, error) {
	for _, f := range this.zr.File {
		if strings.ToLower(f.Name) == path {
			return f.Open()
		}
	}
	return nil, os.ErrNotExist
}

func (this *ZipFolder) Walk(fnWalk FxWalk) error {
	for _, f := range this.zr.File {
		if e := fnWalk(f.Name); e != nil {
			return e
		}
	}
	return nil
}

func (this *ZipFolder) ReadDirNames() ([]string, error) {
	names := make([]string, len(this.zr.File))
	for i, f := range this.zr.File {
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
