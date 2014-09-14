package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"code.google.com/p/go.net/html"
)

const (
	invalid_level = -1
	lowest_level  = 6
	unknown_level = lowest_level + 1
)

type EpubMaker struct {
	folder      VirtualFolder
	book        *Epub
	logger      *log.Logger
	output_path string
	chapter_id  int
	toc         int
	split       int
	bydiv       int
}

func NewEpubMaker(logger *log.Logger) *EpubMaker {
	return &EpubMaker{logger: logger}
}

func (this *EpubMaker) parseBook() (*html.Node, error) {
	f, e := this.folder.OpenFile("book.html")
	if e != nil {
		return nil, e
	}
	defer f.Close()
	return html.Parse(f)
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

func getHeaderNodeLevel(node *html.Node) int {
	if len(node.Data) != 2 || node.Data[0] != 'h' {
		return invalid_level
	}

	level := int(node.Data[1] - '0')
	if level <= 0 || level > lowest_level {
		return invalid_level
	}

	if hasClass(node, "makeepub-not-chapter") {
		return invalid_level
	}

	return level
}

func getNonHeaderNodeLevel(node *html.Node) int {
	attr := findAttribute(node, "class")
	if attr == nil {
		return invalid_level
	}

	for _, class := range strings.Fields(attr.Val) {
		if class == "makeepub-chapter" {
			return unknown_level
		}
		if !strings.HasPrefix(class, "makeepub-chapter-level") {
			continue
		}
		level, e := strconv.Atoi(class[len("makeepub-chapter-level"):])
		if level >= 0 && level <= lowest_level && e == nil {
			return level
		}
	}

	return invalid_level
}

func checkNonHeaderChapter(node *html.Node) (int, string) {
	level := getNonHeaderNodeLevel(node)

	if level == invalid_level {
		return invalid_level, ""
	}

	if level != unknown_level {
		title := getAttributeValue(node, "title", "")
		removeAttribute(node, "title")
		return level, title
	}

	for n := node.NextSibling; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if getNonHeaderNodeLevel(n) != invalid_level {
			return invalid_level, ""
		}
		if level = getHeaderNodeLevel(n); level != invalid_level {
			addClass(n, "makeepub-not-chapter")
			return level, n.FirstChild.Data
		}
	}

	return invalid_level, ""
}

func (this *EpubMaker) checkNewChapter(node *html.Node) *Chapter {
	if node.Type != html.ElementNode {
		return nil
	}

	level, title := invalid_level, ""
	if this.bydiv > 0 {
		level, title = checkNonHeaderChapter(node)
	} else if level = getHeaderNodeLevel(node); level != invalid_level {
		title = node.FirstChild.Data
	}
	if level == invalid_level {
		return nil
	}

	id := findAttribute(node, "id")
	if id == nil {
		node.Attr = append(node.Attr, html.Attribute{Key: "id"})
		id = &node.Attr[len(node.Attr)-1]
	}
	if len(id.Val) == 0 {
		id.Val = fmt.Sprintf("makeepub-chapter-%d", this.chapter_id)
		this.chapter_id++
	}

	return &Chapter{
		Level: level,
		Title: strings.TrimSpace(title),
		Link:  "#" + id.Val,
	}
}

func (this *EpubMaker) checkFullScreenImage(node *html.Node) (string, string) {
	if !this.book.Duokan() {
		return "", ""
	}
	if node.Type != html.ElementNode || node.Data != "img" {
		return "", ""
	}
	fs, src, alt := false, "", ""
	for i := 0; i < len(node.Attr); i++ {
		attr := &node.Attr[i]
		if attr.Key == "class" {
			fs = containsField(attr.Val, "duokan-fullscreen")
		} else if attr.Key == "src" {
			src = attr.Val
		} else if attr.Key == "alt" {
			alt = attr.Val
		}
	}
	if fs {
		return src, alt
	}
	return "", ""
}

func (this *EpubMaker) splitChapter() error {
	root, e := this.parseBook()
	if e != nil {
		return e
	}

	title := findChildNode(root, "title").FirstChild
	nodes := findChildNode(root, "body")
	body := resetBody(nodes)
	chapters := make([]Chapter, 0)

	lastLevel := unknown_level

	for node := nodes.FirstChild; node != nil; node = nodes.FirstChild {
		if isBlankNode(node) {
			nodes.RemoveChild(node)
			continue
		}

		if path, alt := this.checkFullScreenImage(node); len(path) > 0 {
			if body.FirstChild != nil {
				this.saveChapter(root, chapters)
				chapters = make([]Chapter, 0)
				body = resetBody(body)
			}
			this.book.AddFullScreenImage(path, alt, nil)
			lastLevel = unknown_level
			nodes.RemoveChild(node)
			continue
		}

		c := this.checkNewChapter(node)
		if c == nil {
			lastLevel = unknown_level
			nodes.RemoveChild(node)
			body.AppendChild(node)
			continue
		}

		// c.Level > lastLevel means current chapter is a child of last
		// chapter, and there's no text (only chapter names), so merge it into
		// last chapter
		if c.Level <= this.split && c.Level <= lastLevel {
			if body.FirstChild != nil {
				this.saveChapter(root, chapters)
				chapters = make([]Chapter, 0)
				body = resetBody(body)
			}
			title.Data = c.Title
			lastLevel = c.Level
		}

		// level 0 is only for chapter split, will not be added to chapter list
		if c.Level > 0 && c.Level <= this.toc && len(c.Title) > 0 {
			chapters = append(chapters, *c)
		}

		nodes.RemoveChild(node)
		body.AppendChild(node)
	}

	if body.FirstChild != nil {
		this.saveChapter(root, chapters)
	}

	return nil
}

func resetBody(body *html.Node) *html.Node {
	nb := cloneNode(body)
	body.Parent.InsertBefore(nb, body)
	body.Parent.RemoveChild(body)
	return nb
}

func (this *EpubMaker) saveChapter(root *html.Node, chapters []Chapter) error {
	buf := new(bytes.Buffer)
	if e := html.Render(buf, root); e != nil {
		return e
	}
	this.book.AddChapter(chapters, buf.Bytes())
	return nil
}

func (this *EpubMaker) writeLog(msg string) {
	this.logger.Printf("%s: %s\n", this.folder.Name(), msg)
}

func (this *EpubMaker) loadConfig() error {
	rc, e := this.folder.OpenFile("book.ini")
	if e != nil {
		return e
	}

	cfg, e := ParseIni(rc)
	rc.Close()
	if e != nil {
		return e
	}

	this.toc = cfg.GetInt("/book/toc", 2)
	if this.toc < 1 || this.toc > lowest_level {
		this.writeLog("option 'toc' is invalid, will use default value 2.")
		this.toc = 2
	}
	this.split = cfg.GetInt("/split/AtLevel", 1)
	if this.split < 0 || this.split > lowest_level {
		this.writeLog("option 'AtLevel' is invalid, will use default value 1.")
		this.split = 1
	}
	this.bydiv = cfg.GetInt("/split/ByDiv", 0)
	if this.bydiv < 0 || this.bydiv > lowest_level {
		this.writeLog("option 'ByDiv' is invalid, will use default value 0.")
		this.bydiv = 0
	}
	this.output_path = cfg.GetString("/output/path", "")

	s := cfg.GetString("/book/id", "")
	this.book.SetId(s)

	s = cfg.GetString("/book/name", "")
	if len(s) == 0 {
		this.writeLog("book name is empty.")
	}
	this.book.SetName(s)

	s = cfg.GetString("/book/author", "")
	if len(s) == 0 {
		this.writeLog("author name is empty.")
	}
	this.book.SetAuthor(s)

	s = cfg.GetString("/book/publisher", "")
	this.book.SetPublisher(s)

	s = cfg.GetString("/book/description", "")
	this.book.SetDescription(s)

	s = cfg.GetString("/book/language", "zh-CN")
	this.book.SetLanguage(s)

	return nil
}

func (this *EpubMaker) Process(folder VirtualFolder, duokan bool) error {
	this.folder = folder
	this.book = NewEpub(duokan)

	if e := this.loadConfig(); e != nil {
		this.writeLog("failed to open configuration file.")
		return e
	}

	if e := this.splitChapter(); e != nil {
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
	path := this.output_path
	if len(path) == 0 {
		this.writeLog("output path is empty, no file will be created.")
		return nil
	}

	if len(outdir) != 0 {
		_, path = filepath.Split(path)
		path = filepath.Join(outdir, path)
	}

	if e := this.book.Save(path, version); e != nil {
		this.writeLog("failed to create output file.")
		return e
	}

	this.writeLog("output file created at '" + path + "'.")
	return nil
}

func (this *EpubMaker) GetResult(ver int) ([]byte, string, error) {
	path := this.output_path
	if len(path) > 0 {
		_, path = filepath.Split(path)
	} else {
		path = "book.epub"
	}

	data, e := this.book.Build(ver)
	return data, path, e
}

func RunMake() {
	var outdir string
	if len(os.Args) > 2 {
		outdir = os.Args[2]
	}

	duokan := !GetArgumentFlagBool(os.Args[1:], "noduokan")
	ver := EPUB_VERSION_300
	if GetArgumentFlagBool(os.Args[1:], "epub2") {
		ver = EPUB_VERSION_200
	}

	maker := NewEpubMaker(logger)

	if folder, e := OpenVirtualFolder(os.Args[1]); e != nil {
		logger.Fatalf("%s: failed to open source folder/file.\n", os.Args[1])
	} else if maker.Process(folder, duokan) != nil {
		os.Exit(1)
	} else if maker.SaveTo(outdir, ver) != nil {
		os.Exit(1)
	}
}
