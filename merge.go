package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"regexp"
	"sort"
)

func mergeFiles(folder InputFolder, names []string) []byte {
	var (
		reBodyStart = regexp.MustCompile("^[ \t]*<(?i)body(?-i)[^>]*>$")
		reBodyEnd   = regexp.MustCompile("^[ \t]*</(?i)body(?-i)>[ \t]*$")
	)
	buf := new(bytes.Buffer)

	for i, name := range names {
		f, e := folder.OpenFile(name)
		if e != nil {
			log.Fatalf("error reading '%s'.\n", name)
		}

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			l := scanner.Text()
			if i == 0 {
				buf.WriteString(l + "\n")
			}
			if reBodyStart.MatchString(l) {
				break
			}
		}
		if scanner.Err() != nil {
			log.Fatalf("error reading '%s'.\n", name)
		}

		for scanner.Scan() {
			l := scanner.Text()
			if reBodyEnd.MatchString(l) {
				break
			}
			buf.WriteString(l + "\n")
		}
		if scanner.Err() != nil {
			log.Fatalf("error reading '%s'.\n", name)
		}

		f.Close()
	}

	buf.WriteString("</body>\n</html>")
	return buf.Bytes()
}

func RunMerge() {
	checkCommandLine(4)

	folder, e := OpenInputFolder(os.Args[2])
	if e != nil {
		log.Fatalln("failed to open input folder.")
	}
	defer folder.Close()

	names, e := folder.ReadDirNames()
	if e != nil {
		log.Fatal("failed to get input file list.")
	}

	if len(names) == 0 {
		log.Println("input folder is empty.")
		return
	}

	sort.Strings(names)

	data := mergeFiles(folder, names)
	if e = ioutil.WriteFile(os.Args[3], data, 0666); e != nil {
		log.Fatalln("failed to write to output file.\n")
	}
}
