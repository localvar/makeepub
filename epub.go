package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	path_of_mimetype      = "mimetype"
	path_of_toc_ncx       = "toc.ncx"
	path_of_nav_xhtml     = "nav.xhtml"
	path_of_content_opf   = "content.opf"
	path_of_container_xml = "META-INF/container.xml"
	path_of_cover_page    = "cover.html"

	VERSION_200 = iota // epub version 2.0
	VERSION_300        // epub version 3.0

	epub_NORMAL_FILE      = 1 << iota // nomal files
	epub_CONTENT_FILE                 // content files: the chapters
	epub_FULL_SCREEN_PAGE             // full screen pages in content
	epub_INTERNAL_FILE                // internal file, generated automatically in most case

	container_xml = "" +
		"<?xml version=\"1.0\"?>\n" +
		"<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n" +
		"	<rootfiles>\n" +
		"		<rootfile full-path=\"" + path_of_content_opf + "\" media-type=\"application/oebps-package+xml\"/>\n" +
		"	</rootfiles>\n" +
		"</container>"
)

var (
	media_types = map[string]string{
		".html":  "application/xhtml+xml",
		".htm":   "application/xhtml+xml",
		".css":   "text/css",
		".txt":   "text/plain",
		".xml":   "text/xml",
		".xhtml": "application/xhtml+xml",
		".ncx":   "application/x-dtbncx+xml",
		".jpg":   "image/jpeg",
		".jpeg":  "image/jpeg",
		".gif":   "image/gif",
		".png":   "image/png",
		".bmp":   "image/bmp",
		".otf":   "application/x-font-opentype",
		".ttf":   "application/x-font-ttf",
	}
)

func getMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	if mt, ok := media_types[ext]; ok {
		return mt
	}
	return "application/octet-stream"
}

////////////////////////////////////////////////////////////////////////////////
// helper class, epub compressor

type epubCompressor struct {
	zip *zip.Writer
	buf *bytes.Buffer
}

func (this *epubCompressor) init() error {
	this.buf = new(bytes.Buffer)
	this.zip = zip.NewWriter(this.buf)

	header := &zip.FileHeader{
		Name:   path_of_mimetype,
		Method: zip.Store,
	}
	w, e := this.zip.CreateHeader(header)
	if e == nil {
		_, e = w.Write([]byte("application/epub+zip"))
	}
	return e
}

func (this *epubCompressor) addFile(path string, data []byte) error {
	w, e := this.zip.Create(path)
	if e == nil {
		_, e = w.Write(data)
	}
	return e
}

func (this *epubCompressor) close() error {
	return this.zip.Close()
}

func (this *epubCompressor) result() []byte {
	return this.buf.Bytes()
}

////////////////////////////////////////////////////////////////////////////////

type Chapter struct {
	Level int
	Title string
	Link  string
}

type File struct {
	Path     string
	Data     []byte
	Attr     int
	Chapters []Chapter
}

type Epub struct {
	id     string
	name   string
	author string
	cover  string
	files  []*File
}

func NewEpub() *Epub {
	this := new(Epub)
	this.files = make([]*File, 0, 256)
	return this
}

func (this *Epub) Id() string {
	if len(this.id) == 0 {
		this.SetId("")
	}
	return this.id
}

func (this *Epub) SetId(id string) {
	if len(id) == 0 {
		h, _ := os.Hostname()
		t := uint32(time.Now().Unix())
		id = fmt.Sprintf("%s-book-%08x", h, t)
	}
	this.id = id
}

func (this *Epub) Name() string {
	return this.name
}

func (this *Epub) SetName(name string) {
	this.name = name
}

func (this *Epub) Author() string {
	return this.author
}

func (this *Epub) SetAuthor(author string) {
	this.author = author
}

func (this *Epub) SetCoverImage(path string) {
	this.cover = filepath.ToSlash(path)
}

func (this *Epub) AddFile(path string, data []byte) {
	path = filepath.ToSlash(path)
	if strings.ToLower(path) == path_of_mimetype {
		return
	}
	f := &File{
		Path: path,
		Data: data,
	}
	if path == path_of_cover_page ||
		path == path_of_content_opf ||
		path == path_of_toc_ncx ||
		path == strings.ToLower(path_of_container_xml) {
		f.Attr = epub_INTERNAL_FILE
	}
	this.files = append(this.files, f)
}

func generateImagePage(path string) []byte {
	path = filepath.ToSlash(path)
	s := fmt.Sprintf(""+
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
		"<!DOCTYPE html>"+
		"<html xmlns=\"http://www.w3.org/1999/xhtml\">\n"+
		"<head>\n"+
		"	<title></title>\n"+
		"</head>\n"+
		"<body>\n"+
		"	<p><img alt=\"%s\" src=\"%s\"/></p>\n"+
		"</body>\n"+
		"</html>\n", path, path)
	return []byte(s)
}

func (this *Epub) AddFullScreenImage(path string) {
	f := &File{
		Path: fmt.Sprintf("full_scrn_img_%04d.html", len(this.files)),
		Data: generateImagePage(path),
		Attr: epub_CONTENT_FILE | epub_FULL_SCREEN_PAGE,
	}
	this.files = append(this.files, f)
}

func (this *Epub) AddChapter(chapters []Chapter, data []byte) {
	f := &File{
		Path:     fmt.Sprintf("chapter_%04d.html", len(this.files)),
		Data:     data,
		Attr:     epub_CONTENT_FILE,
		Chapters: chapters,
	}
	this.files = append(this.files, f)
}

func (this *Epub) generateContentOpf() []byte {
	buf := new(bytes.Buffer)

	fmt.Fprintf(buf, ""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"uuid_id\">\n"+
		"	<metadata xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:opf=\"http://www.idpf.org/2007/opf\" xmlns:dcterms=\"http://purl.org/dc/terms/\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n"+
		"		<dc:language>zh</dc:language>\n"+
		"		<dc:creator>%s</dc:creator>\n"+
		"		<meta name=\"cover\" content=\"%s\"/>\n"+
		"		<meta property=\"dcterms:modified\">%s</meta>\n"+
		"		<dc:title>%s</dc:title>\n"+
		"		<dc:identifier id=\"uuid_id\">%s</dc:identifier>\n"+
		"	</metadata>\n"+
		"	<manifest>\n"+
		"		<item properties=\"nav\" id=\"ncx\" href=\""+path_of_toc_ncx+"\" media-type=\"application/xhtml+xml\"/>\n",
		this.Author(),
		this.cover,
		time.Now().UTC().Format(time.RFC3339),
		this.Name(),
		this.Id(),
	)

	if len(this.cover) > 0 {
		buf.WriteString("		<item href=\"" + path_of_cover_page + "\" id=\"cover\" media-type=\"application/xhtml+xml\"/>\n")
	}

	for i, f := range this.files {
		if (f.Attr & epub_INTERNAL_FILE) != 0 {
			continue
		}
		fmt.Fprintf(buf,
			"		<item href=\"%s\" id=\"item%04d\" media-type=\"%s\"/>\n",
			f.Path,
			i,
			getMediaType(f.Path),
		)
	}

	buf.WriteString("" +
		"	</manifest>\n" +
		"	<spine>\n",
	)

	if len(this.cover) > 0 {
		buf.WriteString("		<itemref idref=\"cover\" linear=\"no\" properties=\"duokan-page-fullscreen\"/>\n")
	}

	for i, f := range this.files {
		if (f.Attr & epub_CONTENT_FILE) == 0 {
			continue
		}
		fmt.Fprintf(buf, "		<itemref idref=\"item%04d\" linear=\"yes\"", i)
		if (f.Attr & epub_FULL_SCREEN_PAGE) == 0 {
			buf.WriteString("/>\n")
		} else {
			buf.WriteString(" properties=\"duokan-page-fullscreen\"/>\n")
		}
	}

	buf.WriteString("" +
		"	</spine>\n" +
		"</package>",
	)

	return buf.Bytes()
}

func (this *Epub) Depth() int {
	d := 0
	for _, f := range this.files {
		for _, c := range f.Chapters {
			if c.Level > d {
				d = c.Level
			}
		}
	}
	return d
}

func (this *Epub) generateTocNcx() []byte {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf, ""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<ncx xmlns=\"http://www.daisy.org/z3986/2005/ncx/\" version=\"2005-1\" xml:lang=\"zho\">\n"+
		"	<head>\n"+
		"		<meta content=\"%s\" name=\"dtb:uid\"/>\n"+
		"		<meta content=\"%d\" name=\"dtb:depth\"/>\n"+
		"		<meta content=\"0\" name=\"dtb:totalPageCount\"/>\n"+
		"		<meta content=\"0\" name=\"dtb:maxPageNumber\"/>\n"+
		"		<meta name=\"builder\" content=\"makeepub\"/>\n"+
		"	</head>\n"+
		"	<docTitle><text>%s</text></docTitle>\n"+
		"	<docAuthor><text>%s</text></docAuthor>"+
		"	<navMap>\n",
		this.Id(),
		this.Depth(),
		this.Name(),
		this.Author(),
	)

	depth, playorder := 0, 1
	for _, f := range this.files {
		if (f.Attr & epub_CONTENT_FILE) == 0 {
			continue
		}
		for _, c := range f.Chapters {
			if c.Level == depth {
				buf.WriteString("</navPoint>\n")
			} else if c.Level > depth {
				depth = c.Level
			} else {
				for c.Level <= depth {
					buf.WriteString("</navPoint>\n")
					depth--
				}
			}
			fmt.Fprintf(buf, ""+
				"<navPoint id=\"navPoint-%d\" playOrder=\"%d\">\n"+
				"	<navLabel>\n"+
				"		<text>%s</text>\n"+
				"	</navLabel>\n"+
				"	<content src=\"%s\"/>\n",
				playorder,
				playorder,
				c.Title,
				f.Path+c.Link,
			)
			playorder++
		}
	}
	for depth > 0 {
		buf.WriteString("</navPoint>\n")
		depth--
	}

	buf.WriteString("" +
		"	</navMap>\n" +
		"</ncx>",
	)

	return buf.Bytes()
}

func (this *Epub) Close() error {
	return nil
}

func (this *Epub) Buffer() []byte {
	return nil
}

func (this *Epub) Save(path string) error {
	compressor := epubCompressor{}
	e := compressor.init()
	if e != nil {
		return e
	}

	data := []byte(container_xml)
	if e = compressor.addFile(path_of_container_xml, data); e != nil {
		return e
	}

	data = this.generateContentOpf()
	if e = compressor.addFile(path_of_content_opf, data); e != nil {
		return e
	}

	data = this.generateTocNcx()
	if e = compressor.addFile(path_of_toc_ncx, data); e != nil {
		return e
	}

	if len(this.cover) > 0 {
		data = generateImagePage(this.cover)
		if e = compressor.addFile(path_of_cover_page, data); e != nil {
			return e
		}
	}

	for _, f := range this.files {
		if e = compressor.addFile(f.Path, f.Data); e != nil {
			return e
		}
	}

	if e = compressor.close(); e != nil {
		return e
	}

	f, e := os.Create(path)
	if e == nil {
		_, e = f.Write(compressor.result())
		f.Close()
	}

	return e
}

func (this *Epub) CloseAndSave(path string) error {
	/*	if e := this.Close(); e != nil {
			return e
		}
		return this.Save(path)
	*/
	return this.Save(path)
}

/*
import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	mimetype    = "mimetype"
	toc_ncx     = "toc.ncx"
	path_of_nav_xhtml   = "nav.xhtml"
	content_opf = "content.opf"
	meta_inf    = "META-INF/container.xml"
	cover_html  = "cover.html"
)



func getMediaType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	mt, ok := MediaType[ext]
	if !ok {
		return MediaType[""]
	}
	return mt
}

type Chapter struct {
	Name     string
	Path     string
	Children []Chapter
}

type FileInfo struct {
	Id        string
	Path      string
	MediaType string
	Data      []byte
}

type Epub struct {
	Name     string
	Author   string
	id       string
	cover    string
	maxDepth int
	depth    [6]int
	files    []FileInfo
	buf      *bytes.Buffer
	zip      *zip.Writer
	packOnly bool
}

func (this *Epub) SetName(name string) {
	this.Name = name
}

func (this *Epub) SetAuthor(author string) {
	this.Author = author
}

func (this *Epub) SetId(id string) {
	if len(id) == 0 {
		h, _ := os.Hostname()
		t := uint32(time.Now().Unix())
		id = fmt.Sprintf("%s-book-%08x", h, t)
	}
	this.id = id
}

func (this *Epub) Id() string {
	if len(this.id) == 0 {
		this.SetId("")
	}
	return this.id
}

func (this *Epub) MaxDepth() int {
	return len(this.depth)
}

func (this *Epub) addFileToZip(path string, data []byte) error {
	w, e := this.zip.Create(path)
	if e == nil {
		_, e = w.Write(data)
	}
	return e
}

func NewEpub(packOnly bool) (*Epub, error) {
	this := new(Epub)
	this.files = make([]FileInfo, 0, 256)
	this.buf = new(bytes.Buffer)
	this.zip = zip.NewWriter(this.buf)
	this.packOnly = packOnly

	header := &zip.FileHeader{
		Name:   mimetype,
		Method: zip.Store,
	}
	w, e := this.zip.CreateHeader(header)
	if e != nil {
		return nil, e
	}
	_, e = w.Write([]byte("application/epub+zip"))
	if e != nil {
		return nil, e
	}

	if packOnly {
		return this, nil
	}

	data := []byte("" +
		"<?xml version=\"1.0\"?>\n" +
		"<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n" +
		"	<rootfiles>\n" +
		"		<rootfile full-path=\"" + content_opf + "\" media-type=\"application/oebps-package+xml\"/>\n" +
		"	</rootfiles>\n" +
		"</container>")

	if e = this.addFileToZip(meta_inf, data); e != nil {
		return nil, e
	}

	return this, nil
}

func (this *Epub) AddFile(path string, data []byte) (e error) {
	lp := strings.ToLower(path)
	if lp == mimetype {
		return nil
	}

	if (!this.packOnly) &&
		(lp == toc_ncx || lp == content_opf || lp == strings.ToLower(meta_inf)) {
		return nil
	}

	path = filepath.ToSlash(path)
	if e = this.addFileToZip(path, data); e == nil {
		this.files = append(this.files, FileInfo{path: path})
	}

	return e
}

func (this *Epub) updateDepth(depth int) {
	this.depth[depth-1]++
	if this.maxDepth < depth {
		this.maxDepth = depth
	}
	for ; depth < len(this.depth); depth++ {
		this.depth[depth] = 0
	}
}

func (this *Epub) AddChapter(title string, data []byte, depth int) error {
	this.updateDepth(depth)
	path := ""
	for i := 0; i < depth; i++ {
		path += fmt.Sprintf("%02d.", this.depth[i]-1)
	}
	path += "html"

	e := this.addFileToZip(path, data)
	if e == nil {
		ef := FileInfo{path: path, title: title, depth: depth}
		this.files = append(this.files, ef)
	}

	return e
}

func (this *Epub) SetCoverImage(path string) error {
	if len(this.cover) > 0 {
		return nil
	}
	this.cover = filepath.ToSlash(path)

	s := fmt.Sprintf(""+
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
		"<html xmlns=\"http://www.w3.org/1999/xhtml\">\n"+
		"<head>\n"+
		"	<title></title>\n"+
		"</head>\n"+
		"<body>\n"+
		"	<p><img alt=\"cover\" src=\"%s\"/></p>\n"+
		"</body>\n"+
		"</html>\n", this.cover)
	return this.AddFile(cover_html, []byte(s))
}

func (this *Epub) generateNavDoc() error {
	buf := new(bytes.Buffer)
	s := fmt.Sprintf(""+
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
		"<html xmlns=\"http://www.w3.org/1999/xhtml\" xmlns:epub=\"http://www.idpf.org/2007/ops\">\n"+
		"	<head>\n"+
		"		<title>%s</title>\n"+
		"	</head>\n"+
		"	<body>\n"+
		"		<nav id=\"toc\" epub:type=\"toc\">\n",
		this.Name)
	buf.WriteString(s)

	depth, index := 0, 1
	for _, fi := range this.files {
		if fi.depth == 0 {
			continue
		} else if fi.depth == depth {
			buf.WriteString("</li>\n<li")
		} else if fi.depth > depth {
			// todo: if fi.depth > depth + 1
			buf.WriteString("<ol>\n<li")
			depth = fi.depth
		} else {
			for fi.depth < depth {
				buf.WriteString("</li>\n</ol>\n")
				depth--
			}
			buf.WriteString("</li>\n<li")
		}

		s = fmt.Sprintf(" id=\"chapter_%d\">\n	<a href=\"%s\">%s</a>\n",
			index,
			fi.path,
			fi.title)
		index++
		buf.WriteString(s)
	}

	for depth > 0 {
		buf.WriteString("</li>\n</ol>\n")
		depth--
	}

	buf.WriteString("		</nav>\n	</body>\n</html>")

	return this.addFileToZip(path_of_nav_xhtml, buf.Bytes())
}

func (this *Epub) generateContentOpf() error {
	buf := new(bytes.Buffer)
	s := fmt.Sprintf(""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"uuid_id\">\n"+
		"	<metadata xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:opf=\"http://www.idpf.org/2007/opf\" xmlns:dcterms=\"http://purl.org/dc/terms/\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n"+
		"		<dc:language>zh</dc:language>\n"+
		"		<dc:creator>%s</dc:creator>\n"+
		"		<meta name=\"cover\" content=\"%s\"/>\n"+
		"		<meta property=\"dcterms:modified\">%s</meta>\n"+
		"		<dc:title>%s</dc:title>\n"+
		"		<dc:identifier id=\"uuid_id\">%s</dc:identifier>\n"+
		"	</metadata>\n"+
		"	<manifest>\n",
		this.Author,
		this.cover,
		time.Now().UTC().Format(time.RFC3339),
		this.Name,
		this.Id(),
	)
	buf.WriteString(s)

	for i, fi := range this.files {
		if fi.path == cover_html {
			continue
		}
		s = fmt.Sprintf("		<item href=\"%s\" id=\"item%04d\" media-type=\"%s\"/>\n",
			fi.path,
			i,
			getMediaType(fi.path),
		)
		buf.WriteString(s)
	}

	if len(this.cover) > 0 {
		buf.WriteString("		<item href=\"" + cover_html + "\" id=\"cover\" media-type=\"application/xhtml+xml\"/>\n")
	}
	buf.WriteString("" +
		"		<item properties=\"nav\" id=\"ncx\" href=\"" + path_of_nav_xhtml + "\" media-type=\"application/xhtml+xml\"/>\n" +
		"	</manifest>\n" +
		"	<spine>\n")
	if len(this.cover) > 0 {
		buf.WriteString("		<itemref idref=\"cover\" linear=\"no\" properties=\"duokan-page-fullscreen\"/>\n")
	}
	for i, fi := range this.files {
		if fi.depth > 0 {
			s = fmt.Sprintf("		<itemref idref=\"item%04d\" linear=\"yes\"/>\n", i)
			buf.WriteString(s)
		}
	}

	buf.WriteString("	</spine>\n</package>")

	return this.addFileToZip(content_opf, buf.Bytes())
}

func (this *Epub) generateTocNcx() error {
	buf := new(bytes.Buffer)
	s := fmt.Sprintf(""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<ncx xmlns=\"http://www.daisy.org/z3986/2005/ncx/\" version=\"2005-1\" xml:lang=\"zho\">\n"+
		"	<head>\n"+
		"		<meta content=\"%s\" name=\"dtb:uid\"/>\n"+
		"		<meta content=\"%d\" name=\"dtb:depth\"/>\n"+
		"		<meta content=\"0\" name=\"dtb:totalPageCount\"/>\n"+
		"		<meta content=\"0\" name=\"dtb:maxPageNumber\"/>\n"+
		"	</head>\n"+
		"	<docTitle>\n"+
		"		<text>%s</text>\n"+
		"	</docTitle>\n"+
		"	<navMap>\n",
		this.Id(),
		this.maxDepth,
		this.Name,
	)
	buf.WriteString(s)

	depth, playorder := 0, 1
	for _, fi := range this.files {
		if fi.depth == 0 {
			continue
		} else if fi.depth == depth {
			buf.WriteString("</navPoint>\n")
		} else if fi.depth > depth {
			// todo: if fi.depth > depth + 1
			depth = fi.depth
		} else {
			for fi.depth <= depth {
				buf.WriteString("</navPoint>\n")
				depth--
			}
		}

		s = fmt.Sprintf(""+
			"<navPoint id=\"navPoint-%d\" playOrder=\"%d\">\n"+
			"	<navLabel>\n"+
			"		<text>%s</text>\n"+
			"	</navLabel>\n"+
			"	<content src=\"%s\"/>\n",
			playorder,
			playorder,
			fi.title,
			fi.path,
		)
		playorder++
		buf.WriteString(s)
	}

	for depth > 0 {
		buf.WriteString("</navPoint>\n")
		depth--
	}

	buf.WriteString("	</navMap>\n</ncx>")

	return this.addFileToZip(toc_ncx, buf.Bytes())
}

func (this *Epub) generateContentOpf2() error {
	buf := new(bytes.Buffer)
	s := fmt.Sprintf(""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"uuid_id\">\n"+
		"	<metadata xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:opf=\"http://www.idpf.org/2007/opf\" xmlns:dcterms=\"http://purl.org/dc/terms/\" xmlns:calibre=\"http://calibre.kovidgoyal.net/2009/metadata\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n"+
		"		<dc:language>zh</dc:language>\n"+
		"		<dc:creator opf:role=\"aut\">%s</dc:creator>\n"+
		"		<meta name=\"cover\" content=\"%s\"/>\n"+
		"		<dc:date>%s</dc:date>\n"+
		"		<dc:title>%s</dc:title>\n"+
		"		<dc:identifier id=\"uuid_id\">%s</dc:identifier>\n"+
		"	</metadata>\n"+
		"	<manifest>\n",
		this.Author,
		this.cover,
		time.Now().Format(time.RFC3339),
		this.Name,
		this.Id(),
	)
	buf.WriteString(s)

	for i, fi := range this.files {
		if fi.path == cover_html {
			continue
		}
		s = fmt.Sprintf("		<item href=\"%s\" id=\"item%04d\" media-type=\"%s\"/>\n",
			fi.path,
			i,
			getMediaType(fi.path),
		)
		buf.WriteString(s)
	}

	if len(this.cover) > 0 {
		buf.WriteString("		<item href=\"" + cover_html + "\" id=\"cover\" media-type=\"application/xhtml+xml\"/>\n")
	}
	buf.WriteString("" +
		"		<item href=\"" + toc_ncx + "\" media-type=\"application/x-dtbncx+xml\" id=\"ncx\"/>\n" +
		"	</manifest>\n" +
		"	<spine toc=\"ncx\">\n")
	if len(this.cover) > 0 {
		buf.WriteString("		<itemref idref=\"cover\" linear=\"no\" properties=\"duokan-page-fullscreen\"/>\n")
	}
	for i, fi := range this.files {
		if fi.depth > 0 {
			s = fmt.Sprintf("		<itemref idref=\"item%04d\" linear=\"yes\"/>\n", i)
			buf.WriteString(s)
		}
	}

	buf.WriteString("	</spine>\n	<guide>\n")
	if len(this.cover) > 0 {
		buf.WriteString("		<reference href=\"" + cover_html + "\" type=\"cover\" title=\"Cover\"/>\n")
	}
	buf.WriteString("	</guide>\n</package>")

	return this.addFileToZip(content_opf, buf.Bytes())
}

func (this *Epub) Close() error {
	if !this.packOnly {
		//if e := this.generateTocNcx(); e != nil {
		if e := this.generateNavDoc(); e != nil {
			return e
		}
		if e := this.generateContentOpf(); e != nil {
			return e
		}
	}
	return this.zip.Close()
}

func (this *Epub) Buffer() []byte {
	return this.buf.Bytes()
}

func (this *Epub) Save(path string) error {
	if f, e := os.Create(path); e != nil {
		return e
	} else {
		_, e = f.Write(this.Buffer())
		f.Close()
		return e
	}
}

func (this *Epub) CloseAndSave(path string) error {
	if e := this.Close(); e != nil {
		return e
	}
	return this.Save(path)
}
*/
