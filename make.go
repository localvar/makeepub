package main

import (
	"bufio"
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	reHeader = regexp.MustCompile("^[ \t]*<[hH]([1-6])[^>]*>([^<]*)</[hH]([1-6])>[ \t]*$")
	reBody   = regexp.MustCompile("^[ \t]*<(?i)body(?-i)[^>]*>$")
)

func setCoverPage(book *Epub, folder InputFolder) error {
	f, e := folder.OpenFile("cover.html")
	if e != nil {
		return e
	}
	defer f.Close()

	if data, e := ioutil.ReadAll(f); e == nil {
		book.SetCoverPage("cover.html", data)
	}

	return e
}

func addFilesToBook(book *Epub, folder InputFolder) error {
	walk := func(path string) error {
		p := strings.ToLower(path)
		if p == "book.ini" || p == "book.html" || p == "cover.html" {
			return nil
		}

		rc, e := folder.OpenFile(path)
		if e != nil {
			return e
		}
		defer rc.Close()
		data, e := ioutil.ReadAll(rc)
		if e != nil {
			return e
		}

		return book.AddFile(path, data)
	}

	return folder.Walk(walk)
}

func checkNewChapter(l string) (depth int, title string) {
	if m := reHeader.FindStringSubmatch(l); m != nil && m[1] == m[3] {
		depth = int(m[1][0] - '0')
		title = m[2]
	}
	return
}

func addChaptersToBook(book *Epub, folder InputFolder, maxDepth int) error {
	f, e := folder.OpenFile("book.html")
	if e != nil {
		return e
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)

	header := ""
	for scanner.Scan() {
		l := scanner.Text()
		header += l + "\n"
		if reBody.MatchString(l) {
			break
		}
	}
	if e = scanner.Err(); e != nil {
		return e
	}

	buf := new(bytes.Buffer)
	depth, title := 1, ""
	for scanner.Scan() {
		l := scanner.Text()
		if nd, nt := checkNewChapter(l); nd > 0 && nd <= maxDepth {
			if buf.Len() > 0 {
				buf.WriteString("	</body>\n</html>")
				if e = book.AddChapter(title, buf.Bytes(), depth); e != nil {
					return e
				}
				buf.Reset()
			}
			depth, title = nd, nt
			buf.WriteString(header)
		}

		buf.WriteString(l + "\n")
	}
	if e = scanner.Err(); e != nil {
		return e
	}

	if buf.Len() > 0 {
		e = book.AddChapter(title, buf.Bytes(), depth)
	}

	return nil
}

func loadConfig(folder InputFolder) (*Config, error) {
	rc, e := folder.OpenFile("book.ini")
	if e != nil {
		return nil, e
	}
	defer rc.Close()
	return ParseIni(rc)
}

func MakeBook(input string, outdir string) error {
	folder, e := OpenInputFolder(input)
	if e != nil {
		log.Printf("%s: failed to open source folder/file.\n", input)
		return e
	}
	defer folder.Close()

	cfg, e := loadConfig(folder)
	if e != nil {
		log.Printf("%s: failed to open 'book.ini'.\n", input)
		return e
	}

	book, e := NewEpub(false)
	if e != nil {
		log.Printf("%s: failed to create epub book.\n", input)
		return e
	}

	s := cfg.GetString("/book/id", "")
	book.SetId(s)

	s = cfg.GetString("/book/name", "")
	if len(s) == 0 {
		log.Printf("%s: book name is empty.\n", input)
	}
	book.SetName(s)

	s = cfg.GetString("/book/author", "")
	if len(s) == 0 {
		log.Printf("%s: author name is empty.\n", input)
	}
	book.SetAuthor(s)

	if e = setCoverPage(book, folder); e != nil {
		log.Printf("%s: failed to set cover page.\n", input)
		return e
	}

	if e = addFilesToBook(book, folder); e != nil {
		log.Printf("%s: failed to add files to book.\n", input)
		return e
	}

	depth := cfg.GetInt("/book/depth", 1)
	if depth < 1 || depth > book.MaxDepth() {
		log.Printf("%s: invalid 'depth' value, reset to '1'.\n", input)
		depth = 1
	}
	if e = addChaptersToBook(book, folder, depth); e != nil {
		log.Printf("%s: failed to add chapters to book.\n", input)
		return e
	}

	if s = cfg.GetString("/output/path", ""); len(s) == 0 {
		log.Printf("%s: output path is empty, no file will be created.\n", input)
		return nil
	}

	if len(outdir) != 0 {
		_, s = filepath.Split(s)
		s = filepath.Join(outdir, s)
	}
	if e = book.Save(s); e != nil {
		log.Printf("%s: failed to create output file.\n", input)
		return e
	}

	return nil
}

func RunMake() {
	outdir := ""
	if len(os.Args) > 2 {
		outdir = os.Args[2]
	}

	if MakeBook(os.Args[1], outdir) != nil {
		os.Exit(1)
	}
}
