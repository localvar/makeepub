package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

////////////////////////////////////////////////////////////////////////////////

type FxWalk func(path string) error

type IFileRepo interface {
	OpenFile(path string) (io.ReadCloser, error)
	Walk(fnWalk FxWalk) error
	Name() string
	Close()
}

////////////////////////////////////////////////////////////////////////////////

type FileRepo struct {
	base string
}

func NewFileRepo(path string) *FileRepo {
	fs := new(FileRepo)
	fs.base = path
	return fs
}

func (fs *FileRepo) Name() string {
	return fs.base
}

func (fs *FileRepo) OpenFile(path string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(fs.base, path))
}

func (fs *FileRepo) Walk(fnWalk FxWalk) error {
	walk := func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		path, _ = filepath.Rel(fs.base, path)
		return fnWalk(path)
	}

	return filepath.Walk(fs.base, walk)
}

func (fs *FileRepo) Close() {
}

////////////////////////////////////////////////////////////////////////////////

type ZipFileRepo struct {
	path string
	zrc  *zip.ReadCloser
}

func NewZipFileRepo(path string) (*ZipFileRepo, error) {
	rc, e := zip.OpenReader(path)
	if e != nil {
		return nil, e
	}
	zs := new(ZipFileRepo)
	zs.zrc = rc
	zs.path = path
	return zs, nil
}

func (zs *ZipFileRepo) Name() string {
	return zs.path
}

func (zs *ZipFileRepo) Close() {
	zs.zrc.Close()
}

func (zs *ZipFileRepo) OpenFile(path string) (io.ReadCloser, error) {
	for _, f := range zs.zrc.File {
		if strings.ToLower(f.Name) == path {
			return f.Open()
		}
	}
	return nil, os.ErrNotExist
}

func (zs *ZipFileRepo) Walk(fnWalk FxWalk) error {
	for _, f := range zs.zrc.File {
		if e := fnWalk(f.Name); e != nil {
			return e
		}
	}
	return nil
}

////////////////////////////////////////////////////////////////////////////////	

var (
	reHeader = regexp.MustCompile("^[ \t]*<[hH]([1-6])[^>]*>([^<]*)</[hH]([1-6])>[ \t]*$")
	reBody   = regexp.MustCompile("^[ \t]*<(?i)body(?-i)[^>]*>$")
)

func setCoverPage(book *Epub, fr IFileRepo) error {
	f, e := fr.OpenFile("cover.html")
	if e != nil {
		return e
	}
	defer f.Close()

	if data, e := ioutil.ReadAll(f); e == nil {
		book.SetCoverPage("cover.html", data)
	}

	return e
}

func addFilesToBook(book *Epub, fr IFileRepo) error {
	walk := func(path string) error {
		p := strings.ToLower(path)
		if p == "book.ini" || p == "book.html" || p == "cover.html" {
			return nil
		}

		rc, e := fr.OpenFile(path)
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

	return fr.Walk(walk)
}

func checkNewChapter(l string) (depth int, title string) {
	if m := reHeader.FindStringSubmatch(l); m != nil && m[1] == m[3] {
		depth = int(m[1][0] - '0')
		title = m[2]
	}
	return
}

func addChaptersToBook(book *Epub, fr IFileRepo, maxDepth int) error {
	f, e := fr.OpenFile("book.html")
	if e != nil {
		return e
	}
	defer f.Close()
	br := bufio.NewReader(f)

	header := ""
	for {
		s, _, e := br.ReadLine()
		if e != nil {
			return e
		}
		l := string(s)
		header += l + "\n"
		if reBody.MatchString(l) {
			break
		}
	}

	buf := new(bytes.Buffer)
	depth, title := 1, ""
	for {
		s, _, e := br.ReadLine()
		if e == io.EOF {
			break
		}
		l := string(s)
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

	if buf.Len() > 0 {
		e = book.AddChapter(title, buf.Bytes(), depth)
	}

	return nil
}

func loadConfig(fr IFileRepo) (*Config, error) {
	rc, e := fr.OpenFile("book.ini")
	if e != nil {
		return nil, e
	}
	defer rc.Close()
	return ParseIni(rc)
}

func makeBook(input string, output string) error {
	fr, e := createFileRepo(input)
	if e != nil {
		log.Printf("%s : failed to open source folder/file.\n", input)
		return e
	}
	defer fr.Close()

	cfg, e := loadConfig(fr)
	if e != nil {
		log.Printf("%s : failed to open 'book.ini'.\n", input)
		return e
	}

	s := cfg.GetString("/book/id", "")
	book, e := NewEpub(s)
	if e != nil {
		log.Printf("%s : failed to create epub book.\n", input)
		return e
	}

	s = cfg.GetString("/book/name", "")
	if len(s) == 0 {
		log.Printf("%s : book name is empty.\n", input)
	}
	book.SetName(s)

	s = cfg.GetString("/book/author", "")
	if len(s) == 0 {
		log.Printf("%s : author name is empty.\n", input)
	}
	book.SetAuthor(s)

	if e = setCoverPage(book, fr); e != nil {
		log.Printf("%s : failed to set cover page.\n", input)
		return e
	}

	if e = addFilesToBook(book, fr); e != nil {
		log.Printf("%s : failed to add files to book.\n", input)
		return e
	}

	depth := cfg.GetInt("/book/depth", 1)
	if depth < 1 || depth > book.MaxDepth() {
		log.Printf("%s : invalid 'depth' value, reset to '1'.\n", input)
		depth = 1
	}
	if e = addChaptersToBook(book, fr, depth); e != nil {
		log.Printf("%s : failed to add chapters to book.\n", input)
		return e
	}

	if len(output) == 0 {
		output = cfg.GetString("/output/path", "")
	}
	if len(output) == 0 {
		log.Printf("%s : output path has not been set.\n", input)
	} else if e = book.Save(output); e != nil {
		fmt.Println(output)
		log.Printf("%s : failed to create output file.\n", input)
		return e
	}

	return nil
}

func createFileRepo(path string) (IFileRepo, error) {
	stat, e := os.Stat(path)
	if e != nil {
		return nil, e
	}

	if stat.IsDir() {
		return NewFileRepo(path), nil
	}

	return NewZipFileRepo(path)
}

func runMake() {
	output := ""
	if len(os.Args) > 2 {
		output = os.Args[2]
	}

	if makeBook(os.Args[1], output) != nil {
		os.Exit(1)
	}
}

func runBatch() {
}

func getBinaryName() string {
	name := os.Args[0]
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			name = name[0:i]
		} else if os.IsPathSeparator(name[i]) {
			name = name[i+1:]
			break
		}
	}
	return name
}

func showUsage() {
	bn := getBinaryName()
	fmt.Printf("Usage: %s <folder>  [output]\n", bn)
	fmt.Printf("       %s <zip> [output]\n", bn)
	fmt.Printf("       %s <-? | -h | -H>\n", bn)
	os.Exit(0)
}

func checkCommandLine(minArg int) {
	if len(os.Args) < minArg {
		log.Fatalf("Invalid command line. See '%s -?'\n", getBinaryName())
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(getBinaryName() + ": ")

	checkCommandLine(2)

	start := time.Now()

	switch os.Args[1] {
	case "-b", "-B":
		runBatch()
	case "-h", "-H", "-?":
		showUsage()
	default:
		runMake()
	}

	log.Println("done, time used:", time.Now().Sub(start).String())
	os.Exit(0)
}
