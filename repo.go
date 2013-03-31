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

type IFileRepo interface {
	OpenFile(path string) (io.ReadCloser, error)
	Walk(fnWalk FxWalk) error
	Name() string
	Close()
}

////////////////////////////////////////////////////////////////////////////////

type FileRepo struct {
	base string
}

func NewFileRepo(path string) *FileRepo {
	repo := new(FileRepo)
	repo.base = path
	return repo
}

func (repo *FileRepo) Name() string {
	return repo.base
}

func (repo *FileRepo) OpenFile(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(repo.base, path))
}

func (repo *FileRepo) Walk(fnWalk FxWalk) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path, _ = filepath.Rel(repo.base, path)
		return fnWalk(path)
	}

	return filepath.Walk(repo.base, walk)
}

func (repo *FileRepo) Close() {
}

////////////////////////////////////////////////////////////////////////////////

type ZipFileRepo struct {
	path string
	zrc  *zip.ReadCloser
}

func NewZipFileRepo(path string) (*ZipFileRepo, error) {
	rc, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	repo := new(ZipFileRepo)
	repo.zrc = rc
	repo.path = path
	return repo, nil
}

func (repo *ZipFileRepo) Name() string {
	return repo.path
}

func (repo *ZipFileRepo) Close() {
	repo.zrc.Close()
}

func (repo *ZipFileRepo) OpenFile(path string) (io.ReadCloser, error) {
	for _, f := range repo.zrc.File {
		if strings.ToLower(f.Name) == path {
			return f.Open()
		}
	}
	return nil, os.ErrNotExist
}

func (repo *ZipFileRepo) Walk(fnWalk FxWalk) error {
	for _, f := range repo.zrc.File {
		if e := fnWalk(f.Name); e != nil {
			return e
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////

func CreateFileRepo(path string) (IFileRepo, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return nil, e
	}

	if stat.IsDir() {
		return NewFileRepo(path), nil
	}

	return NewZipFileRepo(path)
}

////////////////////////////////////////////////////////////////////////////////
