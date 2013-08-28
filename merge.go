package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"os"
	"regexp"
	"sort"
)

func mergeHtml(folder VirtualFolder, names []string) []byte {
	var (
		reBodyStart = regexp.MustCompile("^[ \t]*<(?i)body(?-i)[^>]*>$")
		reBodyEnd   = regexp.MustCompile("^[ \t]*</(?i)body(?-i)>[ \t]*$")
	)
	buf := new(bytes.Buffer)

	for i, name := range names {
		f, e := folder.OpenFile(name)
		if e != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			l := scanner.Bytes()
			if i == 0 {
				buf.Write(l)
				buf.WriteByte('\n')
			}
			if reBodyStart.Match(l) {
				break
			}
		}
		if scanner.Err() != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		for scanner.Scan() {
			l := scanner.Bytes()
			if reBodyEnd.Match(l) {
				break
			}
			buf.Write(l)
			buf.WriteByte('\n')
		}
		if scanner.Err() != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		f.Close()
	}

	buf.WriteString("</body>\n</html>")
	return removeUtf8Bom(buf.Bytes())
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
