package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
)

func RunExtract() {
	checkCommandLineArgumentCount(4)

	zrc, e := zip.OpenReader(os.Args[2])
	if e != nil {
		logger.Fatalln("failed to open input file.")
	}
	defer zrc.Close()

	if e = os.MkdirAll(os.Args[3], os.ModeDir|0666); e != nil {
		logger.Fatalln("failed to create output folder.")
	}

	for _, zf := range zrc.File {
		path := filepath.Join(os.Args[3], zf.Name)

		// skip folders, if it is not empty, will be created during file creation
		if zf.FileInfo().IsDir() {
			continue
		}

		// create the folder if needed, but no need to check error
		dir, _ := filepath.Split(path)
		os.MkdirAll(dir, os.ModeDir|0666)

		rc, e := zf.Open()
		if e != nil {
			logger.Printf("failed to open '%s'.\n", zf.Name)
			continue
		}

		if f, e := os.Create(path); e != nil {
			logger.Printf("failed to create output file '%s'.", zf.Name)
		} else if _, e = io.Copy(f, rc); e != nil {
			logger.Printf("error writing data to '%s'.\n", zf.Name)
		} else {
			f.Close()
		}

		rc.Close()
	}
}
