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
			l := scanner.Text()
			if i == 0 {
				buf.WriteString(l)
				buf.WriteString("\n")
			}
			if reBodyStart.MatchString(l) {
				break
			}
		}
		if scanner.Err() != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		for scanner.Scan() {
			l := scanner.Text()
			if reBodyEnd.MatchString(l) {
				break
			}
			buf.WriteString(l)
			buf.WriteString("\n")
		}
		if scanner.Err() != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		f.Close()
	}

	buf.WriteString("</body>\n</html>")
	return buf.Bytes()
}

func mergeText(folder VirtualFolder, names []string) []byte {
	buf := new(bytes.Buffer)

	for _, name := range names {
		f, e := folder.OpenFile(name)
		if e != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			buf.WriteString(scanner.Text())
			buf.WriteString("\n")
		}
		if scanner.Err() != nil {
			logger.Fatalf("error reading '%s'.\n", name)
		}

		f.Close()
	}

	return buf.Bytes()
}

func RunMerge() {
	checkCommandLine(4)

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
