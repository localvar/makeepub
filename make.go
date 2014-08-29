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

const highest_level = 6

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

func findAttrByName(node *html.Node, name string) *html.Attribute {
	for i := 0; i < len(node.Attr); i++ {
		if node.Attr[i].Key == name {
			return &node.Attr[i]
		}
	}
	return nil
}

func getHeaderNodeLevel(node *html.Node) int {
	if len(node.Data) != 2 || node.Data[0] != 'h' {
		return -1
	}

	if level := int(node.Data[1] - '0'); level > 0 && level <= highest_level {
		return level
	}

	return -1
}

func getDivNodeLevel(node *html.Node) int {
	if node.Data != "div" {
		return -1
	}

	attr := findAttrByName(node, "class")
	if attr == nil {
		return -1
	}

	if attr.Val == "makeepub-chapter" {
		return 0
	}

	s := attr.Val[len("makeepub-chapter-level"):]
	if len(s) != 1 {
		return -1
	}

	if level := int(s[0] - '0'); level > 0 && level <= highest_level {
		return level
	}

	return -1
}

func checkNewDivChapter(node *html.Node) (int, string) {
	level := getDivNodeLevel(node)

	if level == -1 {
		return -1, ""
	}

	// remove all child nodes of this node, but save the first child
	// as it may be used for title
	cn := node.FirstChild
	node.FirstChild, node.LastChild = nil, nil

	if level > 0 {
		title := ""
		if cn != nil {
			title = cn.Data
		}
		return level, title
	}

	for n := node.NextSibling; n != nil; n = n.NextSibling {
		if n.Type != html.ElementNode {
			continue
		}
		if getDivNodeLevel(n) != -1 {
			return -1, ""
		}
		if level = getHeaderNodeLevel(n); level != -1 {
			return level, n.FirstChild.Data
		}
	}

	return -1, ""
}

func (this *EpubMaker) checkNewChapter(node *html.Node, byDiv bool) *Chapter {
	if node.Type != html.ElementNode {
		return nil
	}

	level, title := -1, ""
	if byDiv {
		level, title = checkNewDivChapter(node)
	} else if level = getHeaderNodeLevel(node); level != -1 {
		title = node.FirstChild.Data
	}
	if level == -1 {
		return nil
	}

	id := findAttrByName(node, "id")
	if id == nil {
		node.Attr = append(node.Attr, html.Attribute{Key: "id"})
		id = &node.Attr[len(node.Attr)-1]
	}
	if len(id.Val) == 0 {
		this.chapters++
		id.Val = fmt.Sprintf("makeepub-chapter-%d", this.chapters)
	}

	return &Chapter{
		Level: level,
		Title: strings.TrimSpace(title),
		Link:  "#" + id.Val,
	}
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

func checkFullScreenImage(node *html.Node, duokan bool) (string, string) {
	if (!duokan) || node.Type != html.ElementNode || node.Data != "img" {
		return "", ""
	}
	fs, src, alt := false, "", ""
	for i := 0; i < len(node.Attr); i++ {
		attr := &node.Attr[i]
		if attr.Key == "class" {
			fs = attr.Val == "duokan-fullscreen"
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

func (this *EpubMaker) saveChapter(root *html.Node, chapters []Chapter) error {
	buf := new(bytes.Buffer)
	if e := html.Render(buf, root); e != nil {
		return e
	}
	this.book.AddChapter(chapters, buf.Bytes())
	return nil
}

func (this *EpubMaker) splitChapter(duokan bool) error {
	root, e := this.parseBook()
	if e != nil {
		return e
	}

	toc := this.cfg.GetInt("/book/toc", 2)
	split := this.cfg.GetInt("/split/AtLevel", 1)
	byDiv := this.cfg.GetBool("/split/ByDiv", false)

	title := findNodeByName(root, "title").FirstChild
	nodes := findNodeByName(root, "body")
	body := resetBody(nodes)
	chapters := make([]Chapter, 0)

	lastLevel := highest_level + 1

	for node := nodes.FirstChild; node != nil; node = nodes.FirstChild {
		if isBlankNode(node) {
			nodes.RemoveChild(node)
			continue
		}

		if path, alt := checkFullScreenImage(node, duokan); len(path) > 0 {
			if body.FirstChild != nil {
				this.saveChapter(root, chapters)
				chapters = make([]Chapter, 0)
				body = resetBody(body)
			}
			this.book.AddFullScreenImage(path, alt)
			lastLevel = highest_level + 1
			nodes.RemoveChild(node)
			continue
		}

		c := this.checkNewChapter(node, byDiv)
		if c == nil {
			lastLevel = highest_level + 1
			nodes.RemoveChild(node)
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

		if c.Level <= toc && len(c.Title) > 0 {
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

	s = this.cfg.GetString("/book/publisher", "")
	this.book.SetPublisher(s)

	s = this.cfg.GetString("/book/description", "")
	this.book.SetDescription(s)

	s = this.cfg.GetString("/book/language", "zh-CN")
	this.book.SetLanguage(s)

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
		s = filepath.Join(outdir, s)
	}

	if e := this.book.Save(s, version); e != nil {
		this.writeLog("failed to create output file.")
		return e
	}

	this.writeLog("output file created at '" + s + "'.")
	return nil
}

func (this *EpubMaker) GetResult(ver int) ([]byte, string, error) {
	name := this.cfg.GetString("/output/path", "")
	if len(name) > 0 {
		_, name = filepath.Split(name)
	} else {
		name = "book.epub"
	}

	data, e := this.book.Build(ver)
	return data, name, e
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
