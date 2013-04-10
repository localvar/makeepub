package main

import (
	"archive/zip"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func packMimetype(z *zip.Writer) error {
	header := &zip.FileHeader{
		Name:   "mimetype",
		Method: zip.Store,
	}
	w, e := z.CreateHeader(header)
	if e == nil {
		_, e = w.Write([]byte("application/epub+zip"))
	}
	return e
}

func packFiles(zip *zip.Writer, input string) error {
	repo, e := CreateFileRepo(input)
	if e != nil {
		log.Println("failed to open source folder/file.\n")
		return e
	}
	defer repo.Close()

	walk := func(path string) error {
		p := strings.ToLower(path)
		if p == "mimetype" {
			return nil
		}

		rc, e := repo.OpenFile(path)
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

		w, e := zip.Create(path)
		if e != nil {
			_, e = w.Write(data)
		}

		if e != nil {
			log.Println("failed to pack file: ", path)
		}

		return e
	}

	return repo.Walk(walk)
}

func RunPack() {
	checkCommandLine(4)

	buf := new(bytes.Buffer)
	zip := zip.NewWriter(buf)
	if packMimetype(zip) != nil {
		log.Fatalln("failed to add mimetype")
	}
	if packFiles(zip, os.Args[2]) != nil {
		os.Exit(1)
	}
	zip.Close()

	f, e := os.Create(os.Args[3])
	if e != nil {
		log.Fatalln("failed to create output file: ")
	}
	defer f.Close()
	if _, e = f.Write(buf.Bytes()); e != nil {
		log.Fatalln("failed writing output file")
	}
}
