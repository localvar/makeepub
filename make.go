package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"code.google.com/p/go.net/html"
)

type EpubMaker struct {
	folder   VirtualFolder
	book     *Epub
	logger   *log.Logger
	cfg      *Config
	chapters int
}

func NewEpubMaker(logger *log.Logger) *EpubMaker {
	return &EpubMaker{logger: logger}
}

func (this *EpubMaker) loadConfig() error {
	rc, e := this.folder.OpenFile("book.ini")
	if e != nil {
		return e
	}
	defer rc.Close()
	this.cfg, e = ParseIni(rc)
	return e
}

func (this *EpubMaker) addFilesToBook() error {
	walk := func(path string) error {
		p := strings.ToLower(path)
		if p == "book.ini" || p == "book.html" {
			return nil
		}

		rc, e := this.folder.OpenFile(path)
		if e != nil {
			return e
		}
		defer rc.Close()
		data, e := ioutil.ReadAll(rc)
		if e != nil {
			return e
		}

		if p == "cover.png" || p == "cover.jpg" || p == "cover.gif" {
			this.book.SetCoverImage(p)
		}
		this.book.AddFile(path, data)
		return nil
	}

	return this.folder.Walk(walk)
}

func findNodeByName(root *html.Node, name string) *html.Node {
	for node := root.FirstChild; node != nil; node = node.NextSibling {
		if node.Type != html.ElementNode {
			continue
		}
		if node.Data == name {
			return node
		}
		if n := findNodeByName(node, name); n != nil {
			return n
		}
	}
	return nil
}

func (this *EpubMaker) checkNewChapter(node *html.Node) *Chapter {
	if node.Type != html.ElementNode {
		return nil
	}

	if len(node.Data) != 2 || node.Data[0] != 'h' {
		return nil
	}

	l := int(node.Data[1] - '0')
	if l < 1 || l > 6 {
		return nil
	}

	var id *html.Attribute
	for i := 0; i < len(node.Attr); i++ {
		if node.Attr[i].Key == "id" {
			id = &node.Attr[i]
			break
		}
	}
	if id == nil {
		node.Attr = append(node.Attr, html.Attribute{Key: "id"})
		id = &node.Attr[len(node.Attr)-1]
	}
	if len(id.Val) == 0 {
		this.chapters++
		id.Val = fmt.Sprintf("makeepub-chapter-%d", this.chapters)
	}

	return &Chapter{Level: l, Title: node.FirstChild.Data, Link: "#" + id.Val}
}

func resetBody(body *html.Node) *html.Node {
	nb := &html.Node{
		Type:     body.Type,
		DataAtom: body.DataAtom,
		Data:     body.Data,
		Attr:     make([]html.Attribute, len(body.Attr)),
	}
	copy(nb.Attr, body.Attr)

	body.Parent.InsertBefore(nb, body)
	body.Parent.RemoveChild(body)
	return nb
}

func isBlankNode(node *html.Node) bool {
	if node.Type == html.CommentNode {
		return true
	}
	if node.Type != html.TextNode {
		return false
	}
	return len(strings.Trim(node.Data, "\t\n\r ")) == 0
}

func checkFullScreenImage(node *html.Node, duokan bool) string {
	if (!duokan) || node.Type != html.ElementNode || node.Data != "img" {
		return ""
	}
	fs, src := false, ""
	for i := 0; i < len(node.Attr); i++ {
		attr := &node.Attr[i]
		if attr.Key == "class" {
			fs = attr.Val == "duokan-fullscreen"
		} else if attr.Key == "src" {
			src = attr.Val
		}
	}
	if fs {
		return src
	}
	return ""
}

func (this *EpubMaker) saveChapter(root *html.Node, chapters []Chapter) error {
	buf := new(bytes.Buffer)
	if e := html.Render(buf, root); e != nil {
		return e
	}
	this.book.AddChapter(chapters, buf.Bytes())
	return nil
}

func (this *EpubMaker) splitChapter(duokan bool) error {
	f, e := this.folder.OpenFile("book.html")
	if e != nil {
		return e
	}

	defer f.Close()
	root, e := html.Parse(f)
	if e != nil {
		return e
	}

	split := this.cfg.GetInt("/book/split", 1)
	toc := this.cfg.GetInt("/book/toc", 2)

	title := findNodeByName(root, "title").FirstChild
	nodes := findNodeByName(root, "body")
	body := resetBody(nodes)
	chapters := make([]Chapter, 0)

	lastLevel := 7

	for node := nodes.FirstChild; node != nil; node = nodes.FirstChild {
		nodes.RemoveChild(node)
		if isBlankNode(node) {
			continue
		}

		if src := checkFullScreenImage(node, duokan); len(src) > 0 {
			if body.FirstChild != nil {
				this.saveChapter(root, chapters)
				chapters = make([]Chapter, 0)
				body = resetBody(body)
			}
			this.book.AddFullScreenImage(src)
			lastLevel = 7
			continue
		}

		c := this.checkNewChapter(node)
		if c == nil {
			lastLevel = 7
			body.AppendChild(node)
			continue
		}

		// c.Level > lastLevel means current chapter is a child of last
		// chapter, and there's no text (only chapter names), so merge it into
		// last chapter
		if c.Level <= split && c.Level <= lastLevel {
			if body.FirstChild != nil {
				this.saveChapter(root, chapters)
				chapters = make([]Chapter, 0)
				body = resetBody(body)
			}
			title.Data = c.Title
			lastLevel = c.Level
		}
		if c.Level <= toc {
			chapters = append(chapters, *c)
		}

		body.AppendChild(node)
	}

	if body.FirstChild != nil {
		this.saveChapter(root, chapters)
	}

	return nil
}

func (this *EpubMaker) writeLog(msg string) {
	this.logger.Printf("%s: %s\n", this.folder.Name(), msg)
}

func (this *EpubMaker) initBook(duokan bool) {
	this.book = NewEpub()

	s := this.cfg.GetString("/book/id", "")
	this.book.SetId(s)

	s = this.cfg.GetString("/book/name", "")
	if len(s) == 0 {
		this.writeLog("book name is empty.")
	}
	this.book.SetName(s)

	s = this.cfg.GetString("/book/author", "")
	if len(s) == 0 {
		this.writeLog("author name is empty.")
	}
	this.book.SetAuthor(s)

	this.book.EnableDuokan(duokan)
}

func (this *EpubMaker) Process(folder VirtualFolder, duokan bool) error {
	this.folder = folder

	if e := this.loadConfig(); e != nil {
		this.writeLog("failed to open configuration file.")
		return e
	}

	this.initBook(duokan)

	if e := this.splitChapter(duokan); e != nil {
		this.writeLog("failed to add chapters to book.")
		return e
	}

	if e := this.addFilesToBook(); e != nil {
		this.writeLog("failed to add files to book.")
		return e
	}

	return nil
}

func (this *EpubMaker) SaveTo(outdir string, version int) error {
	s := this.cfg.GetString("/output/path", "")
	if len(s) == 0 {
		this.writeLog("output path is empty, no file will be created.")
		return nil
	}

	if len(outdir) != 0 {
		_, s = filepath.Split(s)
		s = filepath.Join(outdir, s) //"testbook.zip")
	}

	if e := this.book.Save(s, version); e != nil {
		this.writeLog("failed to create output file.")
		return e
	}

	this.writeLog("output file created at '" + s + "'.")
	return nil
}

func (this *EpubMaker) GetResult() ([]byte, string, error) {
	name := this.cfg.GetString("/output/path", "")
	if len(name) > 0 {
		_, name = filepath.Split(name)
	} else {
		name = "book.epub"
	}

	data, e := this.book.Build(VERSION_300)
	return data, name, e
}

func RunMake() {
	var outdir string
	if len(os.Args) > 2 {
		outdir = os.Args[2]
	}

	maker := NewEpubMaker(logger)

	if folder, e := OpenVirtualFolder(os.Args[1]); e != nil {
		logger.Fatalf("%s: failed to open source folder/file.\n", os.Args[1])
	} else if maker.Process(folder, true) != nil {
		os.Exit(1)
	} else if maker.SaveTo(outdir, VERSION_300) != nil {
		os.Exit(1)
	}
}
