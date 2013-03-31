package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

func mergeFiles(base string, names []string) []byte {
	var (
		reBodyStart = regexp.MustCompile("^[ \t]*<(?i)body(?-i)[^>]*>$")
		reBodyEnd   = regexp.MustCompile("^[ \t]*</(?i)body(?-i)>[ \t]*$")
	)
	buf := new(bytes.Buffer)

	for i, name := range names {
		path := filepath.Join(base, name)
		f, e := os.Open(path)
		if e != nil {
			log.Fatalf("error reading '%s'.\n", name)
		}

		br := bufio.NewReader(f)
		for {
			l, _, e := br.ReadLine()
			if e != nil {
				log.Fatalf("error reading '%s'.\n", name)
			}
			s := string(l)
			if i == 0 {
				buf.WriteString(s + "\n")
			}
			if reBodyStart.MatchString(s) {
				break
			}
		}

		for {
			l, _, e := br.ReadLine()
			if e != nil {
				log.Fatalf("error reading '%s'.\n", name)
			}
			s := string(l)
			if reBodyEnd.MatchString(s) {
				break
			}
			buf.WriteString(s + "\n")
		}

		f.Close()
	}

	buf.WriteString("</body>\n</html>")
	return buf.Bytes()
}

func RunMerge() {
	checkCommandLine(4)

	f, e := os.Open(os.Args[2])
	if e != nil {
		log.Fatalln("failed to open input folder.")
	}

	names, e := f.Readdirnames(-1)
	if e != nil {
		log.Fatal("failed to get input file list.")
	}
	f.Close()

	if len(names) == 0 {
		log.Println("input folder is empty.")
		return
	}

	sort.Strings(names)

	data := mergeFiles(os.Args[2], names)
	if e = ioutil.WriteFile(os.Args[3], data, 0666); e != nil {
		log.Fatalln("failed to write to output file.\n")
	}
}
