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
	"code.google.com/p/go.net/html/atom"
)

const (
	lowest_level = iota + 6
	unknown_level

	duokan_fullscreen    = "duokan-fullscreen"
	makeepub_chapter_id  = "makeepub-chapter-%d"
	makeepub_chapter     = "makeepub-chapter"
	makeepub_not_chapter = "makeepub-not-chapter"
	data_chapter_level   = "data-chapter-level"
	data_chapter_title   = "data-chapter-title"
)

type EpubMaker struct {
	folder      VirtualFolder
	book        *Epub
	logger      *log.Logger
	output_path string
	chapter_id  int
	toc         int
	split       int
	by_header   int
	body        *html.Node // 'body' element of the original html
	skip        bool       // skip next header (<h1>,<h2>...)?
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
	root, e := html.Parse(f)
	if e != nil {
		return root, e
	}

	e = fmt.Errorf("invalid 'book.html'")
	if root.Type != html.DocumentNode {
		return root, e
	}

	Html := findDirectChild(root, atom.Html)
	if Html == nil {
		return root, e
	}

	// remove any element after 'html', make sure 'html' is the last
	// child of 'root'
	for node := Html.NextSibling; node != nil; node = Html.NextSibling {
		root.RemoveChild(node)
	}

	head := findDirectChild(Html, atom.Head)
	body := findDirectChild(Html, atom.Body)
	if head == nil || body == nil {
		return root, e
	}

	// make sure 'head' is the first child of 'html' and 'body' is the last
	// child, and remove all other children if exist
	for node := Html.FirstChild; node != nil; node = Html.FirstChild {
		Html.RemoveChild(node)
	}
	Html.AppendChild(head)
	Html.AppendChild(body)

	return root, nil
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

func checkHeaderNode(node *html.Node) *Chapter {
	if len(node.Data) != 2 || node.Data[0] != 'h' {
		return nil
	}

	level := int(node.Data[1] - '0')
	if level <= 0 || level > lowest_level {
		return nil
	}

	title := ""
	if attr := findAttribute(node, data_chapter_title); attr != nil {
		title = attr.Val
	} else if node.FirstChild != nil {
		title = node.FirstChild.Data
	}
	return &Chapter{Level: level, Title: title}
}

func (this *EpubMaker) checkChapterNode(node *html.Node) *Chapter {
	if !hasClass(node, makeepub_chapter) {
		return nil
	}

	// if it has the 'level' attribute, it should have 'title' attribute also
	if attr := findAttribute(node, data_chapter_level); attr != nil {
		level, e := strconv.Atoi(attr.Val)
		if e != nil || level < 0 || level > lowest_level {
			this.writeLog("invalid chapter level '" + attr.Val + "', ignored.")
			return nil
		}
		title := getAttributeValue(node, data_chapter_title, "")
		return &Chapter{Level: level, Title: title}
	}

	// try to find next 'header' element for level & title
	for n := this.body.FirstChild; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if hasClass(n, makeepub_chapter) {
			return nil
		}
		if c := checkHeaderNode(n); c != nil {
			this.skip = true
			return c
		}
	}

	return nil
}

func (this *EpubMaker) checkNewChapter(node *html.Node) *Chapter {
	if node.Type != html.ElementNode {
		return nil
	}

	var c *Chapter = nil
	if c = this.checkChapterNode(node); c == nil {
		if c = checkHeaderNode(node); c == nil {
			return nil
		}
		if this.skip {
			this.skip = false
			return nil
		}
		if c.Level < this.by_header || hasClass(node, makeepub_not_chapter) {
			return nil
		}
	}

	// only chapters in TOC need 'id' to generate Link
	if c.Level <= this.toc {
		id := findAttribute(node, "id")
		if id == nil {
			node.Attr = append(node.Attr, html.Attribute{Key: "id"})
			id = &node.Attr[len(node.Attr)-1]
		}
		if len(id.Val) == 0 {
			id.Val = fmt.Sprintf(makeepub_chapter_id, this.chapter_id)
			this.chapter_id++
		}
		c.Link = "#" + id.Val
	}

	c.Title = strings.TrimSpace(c.Title)
	return c
}

func (this *EpubMaker) checkFullScreenImage(node *html.Node) (string, string) {
	if !this.book.Duokan() {
		return "", ""
	}
	if node.Type != html.ElementNode || node.DataAtom != atom.Img {
		return "", ""
	}
	fs, src, alt := false, "", ""
	for i := 0; i < len(node.Attr); i++ {
		attr := &node.Attr[i]
		if attr.Key == "class" {
			fs = containsField(attr.Val, duokan_fullscreen)
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

	this.body = root.LastChild.LastChild
	body := resetBody(this.body)
	chapters := make([]Chapter, 0)
	lastLevel := unknown_level

	for node := this.body.FirstChild; node != nil; node = this.body.FirstChild {
		this.body.RemoveChild(node)

		if isBlankNode(node) {
			continue
		}

		c := this.checkNewChapter(node)

		if path, alt := this.checkFullScreenImage(node); len(path) > 0 {
			this.saveChapter(root, chapters)
			body = resetBody(body)
			chapters = nil
			lastLevel = unknown_level
			this.saveFullScreenImage(path, alt, c)
			continue
		}

		if c == nil {
			lastLevel = unknown_level
			body.AppendChild(node)
			continue
		}

		// c.Level > lastLevel means current chapter is a child of last
		// chapter, and there's no text (only chapter names), so merge it into
		// last chapter
		if c.Level <= this.split && c.Level <= lastLevel {
			this.saveChapter(root, chapters)
			body = resetBody(body)
			chapters = nil
			lastLevel = c.Level
		}

		// level 0 is only for chapter split, will not be added to chapter list
		if c.Level > 0 && c.Level <= this.toc && len(c.Title) > 0 {
			chapters = append(chapters, *c)
		}

		body.AppendChild(node)
	}

	this.saveChapter(root, chapters)
	return nil
}

func resetBody(body *html.Node) *html.Node {
	nb := cloneNode(body)
	body.Parent.AppendChild(nb)
	body.Parent.RemoveChild(body)
	return nb
}

func (this *EpubMaker) saveFullScreenImage(path, alt string, c *Chapter) {
	chapters := make([]Chapter, 0)
	if c != nil && c.Level > 0 && c.Level <= this.toc && len(c.Title) > 0 {
		chapters = append(chapters, *c)
	}
	this.book.AddFullScreenImage(path, alt, chapters)
}

func (this *EpubMaker) saveChapter(root *html.Node, chapters []Chapter) {
	// only save if there are something in this chapter
	if root.LastChild.LastChild.FirstChild != nil {
		buf := new(bytes.Buffer)
		html.Render(buf, root)
		this.book.AddChapter(chapters, buf.Bytes())
	}
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
	this.by_header = cfg.GetInt("/split/ByHeader", 1)
	if this.by_header < 1 || this.by_header > lowest_level {
		this.writeLog("option 'ByHeader' is invalid, will use default value 1.")
		this.by_header = 1
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
		this.writeLog(e.Error())
		this.writeLog("failed to open configuration file.")
		return e
	}

	if e := this.splitChapter(); e != nil {
		this.writeLog(e.Error())
		this.writeLog("failed to add chapters to book.")
		return e
	}

	if e := this.addFilesToBook(); e != nil {
		this.writeLog(e.Error())
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
