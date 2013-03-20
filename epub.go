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

type epubFile struct {
	path  string
	title string
	depth int
}

type Epub struct {
	Name     string
	Author   string
	Id       string
	cover    string
	maxDepth int
	depth    [6]int
	files    []epubFile
	buf      *bytes.Buffer
	zip      *zip.Writer
}

func (epub *Epub) SetName(name string) {
	epub.Name = name
}

func (epub *Epub) SetAuthor(author string) {
	epub.Author = author
}

func (epub *Epub) MaxDepth() int {
	return len(epub.depth)
}

func NewEpub(id string) (*Epub, error) {
	epub := new(Epub)
	epub.Id = id
	if len(id) == 0 {
		h, _ := os.Hostname()
		t := uint32(time.Now().Unix())
		epub.Id = fmt.Sprintf("%s-book-%08x", h, t)
	}
	epub.files = make([]epubFile, 256)
	epub.buf = new(bytes.Buffer)
	epub.zip = zip.NewWriter(epub.buf)

	header := &zip.FileHeader{
		Name:   mimetype,
		Method: zip.Store,
	}
	w, e := epub.zip.CreateHeader(header)
	if e != nil {
		return nil, e
	}
	_, e = w.Write([]byte("application/epub+zip"))
	if e != nil {
		return nil, e
	}

	w, e = epub.zip.Create(meta_inf)
	if e != nil {
		return nil, e
	}
	_, e = w.Write([]byte("" +
		"<?xml version=\"1.0\"?>\n" +
		"<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n" +
		"	<rootfiles>\n" +
		"		<rootfile full-path=\"" + content_opf + "\" media-type=\"application/oebps-package+xml\"/>\n" +
		"	</rootfiles>\n" +
		"</container>"))
	if e != nil {
		return nil, e
	}

	return epub, nil
}

func (epub *Epub) AddFile(path string, data []byte) error {
	path = strings.ToLower(path)
	if path == mimetype || path == toc_ncx || path == content_opf ||
		path == strings.ToLower(meta_inf) {
		return nil
	}

	w, e := epub.zip.Create(path)
	if e == nil {
		_, e = w.Write(data)
		if e == nil {
			epub.files = append(epub.files, epubFile{path: path})
		}
	}
	return e
}

func (epub *Epub) updateDepth(depth int) {
	epub.depth[depth-1]++
	if epub.maxDepth < depth {
		epub.maxDepth = depth
	}
	for ; depth < len(epub.depth); depth++ {
		epub.depth[depth] = 0
	}
}

func (epub *Epub) AddChapter(title string, data []byte, depth int) error {
	epub.updateDepth(depth)
	path := ""
	for i := 0; i < depth; i++ {
		path += fmt.Sprintf("%02d.", epub.depth[i]-1)
	}
	path += "html"

	w, e := epub.zip.Create(path)
	if e == nil {
		_, e = w.Write(data)
		if e == nil {
			ef := epubFile{path: path, title: title, depth: depth}
			epub.files = append(epub.files, ef)
		}
	}

	return e
}

func (epub *Epub) SetCoverPage(path string, data []byte) error {
	w, e := epub.zip.Create(path)
	if e == nil {
		_, e = w.Write(data)
		if e == nil {
			epub.cover = path
		}
	}
	return e
}

func (epub *Epub) generateTocNcx() error {
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
		epub.Id,
		epub.maxDepth,
		epub.Name,
	)
	buf.WriteString(s)

	depth, playorder := 0, 1
	for index := 0; index < len(epub.files); index++ {
		ef := epub.files[index]
		if ef.depth == 0 {
			continue
		}

		if ef.depth == depth {
			buf.WriteString("</navPoint>\n")
		} else if ef.depth > depth {
			// todo: if ef.depth > depth + 1
			depth = ef.depth
		} else {
			for ef.depth <= depth {
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
			ef.title,
			ef.path,
		)
		playorder++
		buf.WriteString(s)
	}

	for depth > 0 {
		buf.WriteString("</navPoint>\n")
		depth--
	}

	buf.WriteString("	</navMap>\n</ncx>")

	w, e := epub.zip.Create(toc_ncx)
	if e == nil {
		_, e = w.Write(buf.Bytes())
	}

	return e
}

func (epub *Epub) generateContentOpf() error {
	buf := new(bytes.Buffer)
	s := fmt.Sprintf(""+
		"<?xml version='1.0' encoding='utf-8'?>\n"+
		"<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"uuid_id\">\n"+
		"	<metadata xmlns:xsi=\"http://www.w3.org/2001/XMLSchema-instance\" xmlns:opf=\"http://www.idpf.org/2007/opf\" xmlns:dcterms=\"http://purl.org/dc/terms/\" xmlns:calibre=\"http://calibre.kovidgoyal.net/2009/metadata\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n"+
		"		<dc:language>zh</dc:language>\n"+
		"		<dc:creator opf:role=\"aut\">%s</dc:creator>\n"+
		"		<meta name=\"cover\" content=\"cover\"/>\n"+
		"		<dc:date>%s</dc:date>\n"+
		"		<dc:title>%s</dc:title>\n"+
		"		<dc:identifier id=\"uuid_id\">%s</dc:identifier>\n"+
		"	</metadata>\n"+
		"	<manifest>\n",
		epub.Author,
		time.Now().Format(time.RFC3339),
		epub.Name,
		epub.Id,
	)
	buf.WriteString(s)

	for i := 0; i < len(epub.files); i++ {
		ef := epub.files[i]
		s = fmt.Sprintf(""+
			"		<item href=\"%s\" id=\"item%04d\" media-type=\"%s\"/>\n",
			ef.path,
			i,
			getMediaType(ef.path),
		)
		buf.WriteString(s)
	}

	buf.WriteString("" +
		"		<item href=\"" + toc_ncx + "\" media-type=\"application/x-dtbncx+xml\" id=\"ncx\"/>\n" +
		"		<item href=\"" + epub.cover + "\" id=\"cover\" media-type=\"application/xhtml+xml\"/>\n" +
		"	</manifest>\n" +
		"	<spine toc=\"ncx\">\n" +
		"		<itemref idref=\"cover\" linear=\"no\" properties=\"duokan-page-fullscreen\"/>\n")

	for i := 0; i < len(epub.files); i++ {
		ef := epub.files[i]
		if ef.depth == 0 {
			continue
		}
		s = fmt.Sprintf("		<itemref idref=\"item%04d\" linear=\"yes\"/>\n", i)
		buf.WriteString(s)
	}

	buf.WriteString("" +
		"	</spine>\n" +
		"	<guide>\n" +
		"		<reference href=\"" + epub.cover + "\" type=\"cover\" title=\"Cover\"/>\n" +
		"	</guide>\n" +
		"</package>")

	w, e := epub.zip.Create(content_opf)
	if e == nil {
		_, e = w.Write(buf.Bytes())
	}
	return e
}

func (epub *Epub) Save(path string) error {
	if e := epub.generateTocNcx(); e != nil {
		return e
	}
	if e := epub.generateContentOpf(); e != nil {
		return e
	}
	epub.zip.Close()
	f, e := os.Create(path)
	if e != nil {
		return e
	}
	defer f.Close()
	_, e = f.Write(epub.buf.Bytes())
	return e
}
