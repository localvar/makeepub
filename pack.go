package main

import (
	"io/ioutil"
	"log"
	"os"
)

func packFiles(book *Epub, input string) error {
	folder, e := OpenInputFolder(input)
	if e != nil {
		log.Println("failed to open source folder/file.\n")
		return e
	}

	walk := func(path string) error {
		rc, e := folder.OpenFile(path)
		if e != nil {
			log.Println("failed to open file: ", path)
			return e
		}
		defer rc.Close()
		data, e := ioutil.ReadAll(rc)
		if e != nil {
			log.Println("failed reading file: ", path)
			return e
		}

		if e = book.AddFile(path, data); e != nil {
			log.Println("failed to pack file: ", path)
		}

		return e
	}

	return folder.Walk(walk)
}

func RunPack() {
	checkCommandLine(4)

	book, e := NewEpub(true)
	if e != nil {
		log.Fatalln("failed to create epub book.")
	}

	if packFiles(book, os.Args[2]) != nil {
		os.Exit(1)
	}

	if book.CloseAndSave(os.Args[3]) != nil {
		log.Fatalln("failed to create output file: ", os.Args[3])
	}
}
