package main

import (
	"io/ioutil"
	"os"
)

func packFiles(book *Epub, input string) error {
	folder, e := OpenVirtualFolder(input)
	if e != nil {
		logger.Println("failed to open source folder/file.\n")
		return e
	}

	walk := func(path string) error {
		rc, e := folder.OpenFile(path)
		if e != nil {
			logger.Println("failed to open file: ", path)
			return e
		}
		defer rc.Close()
		data, e := ioutil.ReadAll(rc)
		if e != nil {
			logger.Println("failed reading file: ", path)
			return e
		}

		book.AddFile(path, data)
		return e
	}

	return folder.Walk(walk)
}

func RunPack() {
	inpath, outpath := getArg(0, ""), getArg(1, "")
	if len(inpath) == 0 || len(outpath) == 0 {
		onCommandLineError()
	}

	book := NewEpub(false)

	if packFiles(book, inpath) != nil {
		os.Exit(1)
	}

	if book.Save(outpath, EPUB_VERSION_NONE) != nil {
		logger.Fatalln("failed to create output file: ", outpath)
	}
}

func init() {
	AddCommandHandler("p", RunPack)
}
