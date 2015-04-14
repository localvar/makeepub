package main

import (
	"bytes"
	"io/ioutil"
	"sort"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
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

		b := findFirstChild(doc, atom.Body)
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
	inpath, outpath := getArg(0, ""), getArg(1, "")
	if len(inpath) == 0 || len(outpath) == 0 {
		onCommandLineError()
	}

	folder, e := OpenVirtualFolder(inpath)
	if e != nil {
		logger.Fatalf("failed to open '%s'.\n", inpath)
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
	if flag := getFlag(0); flag[1] == 'h' || flag[1] == 'H' {
		data = mergeHtml(folder, names)
	} else {
		data = mergeText(folder, names)
	}

	if e = ioutil.WriteFile(outpath, data, 0666); e != nil {
		logger.Fatalln("failed to write to output file.\n")
	}
}

func init() {
	AddCommandHandler("mh", RunMerge)
	AddCommandHandler("mt", RunMerge)
}
