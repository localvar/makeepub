package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"sort"

	"code.google.com/p/go.net/html"
)

func mergeHtml(folder VirtualFolder, names []string) []byte {
	var result *html.Node = nil
	var body *html.Node = nil

	for _, name := range names {
		f, e := folder.OpenFile(name)
		if e != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		doc, e := html.Parse(f)
		f.Close()
		if e != nil {
			logger.Fatalf("error parsing '%s'.\n", name)
		}

		b := findNodeByName(doc, "body")
		if b == nil {
			logger.Fatalf("'%s' has no 'body' element.\n", name)
		}

		if body == nil {
			result = doc
			body = b
			continue
		}

		for n := b.FirstChild; n != nil; n = b.FirstChild {
			b.RemoveChild(n)
			body.AppendChild(n)
		}
	}

	buf := new(bytes.Buffer)
	if e := html.Render(buf, result); e != nil {
		logger.Fatalf("failed render result for '%s'.\n", folder.Name())
	}

	return buf.Bytes()
}

func mergeText(folder VirtualFolder, names []string) []byte {
	buf := new(bytes.Buffer)

	for _, name := range names {
		f, e := folder.OpenFile(name)
		if e != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		data, e := ioutil.ReadAll(f)
		if e != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		buf.Write(removeUtf8Bom(data))
		buf.WriteByte('\n')

		f.Close()
	}

	return buf.Bytes()
}

func RunMerge() {
	CheckCommandLineArgumentCount(4)

	folder, e := OpenVirtualFolder(os.Args[2])
	if e != nil {
		logger.Fatalln("failed to open input folder.")
	}

	names, e := folder.ReadDirNames()
	if e != nil {
		logger.Fatal("failed to get input file list.")
	}

	if len(names) == 0 {
		logger.Println("input folder is empty.")
		return
	}

	sort.Strings(names)

	var data []byte
	if os.Args[1][2] == 'h' || os.Args[1][2] == 'H' {
		data = mergeHtml(folder, names)
	} else {
		data = mergeText(folder, names)
	}

	if e = ioutil.WriteFile(os.Args[3], data, 0666); e != nil {
		logger.Fatalln("failed to write to output file.\n")
	}
}

func init() {
	AddCommandHandler("mh", RunMerge)
	AddCommandHandler("mt", RunMerge)
}
