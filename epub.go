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
	mimetype    = "mimetype"
	toc_ncx     = "toc.ncx"
	content_opf = "content.opf"
	meta_inf    = "META-INF/container.xml"
	cover_html  = "cover.html"
)

var (
	MediaType = map[string]string{
		"":       "application/octet-stream",
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
	mt, ok := MediaType[ext]
	if !ok {
		return MediaType[""]
	}
	return mt
}

type FileInfo struct {
	path  string
	title string
	depth int
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
	this.cover = path

	s := fmt.Sprintf(""+
		"<?xml version=\"1.0\" encoding=\"utf-8\" standalone=\"no\"?>"+
		"<!DOCTYPE html PUBLIC \"-//W3C//DTD XHTML 1.1//EN\" \"http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd\">"+
		"<html xmlns=\"http://www.w3.org/1999/xhtml\">"+
		"<head>"+
		"	<title></title>"+
		"</head>"+
		"<body>"+
		"	<p><img alt=\"cover\" src=\"%s\"/></p>"+
		"</body>"+
		"</html>", path)
	return this.AddFile(cover_html, []byte(s))
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

func (this *Epub) generateContentOpf() error {
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
		s = fmt.Sprintf(""+
			"		<item href=\"%s\" id=\"item%04d\" media-type=\"%s\"/>\n",
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
		if e := this.generateTocNcx(); e != nil {
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
