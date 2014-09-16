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

	EPUB_VERSION_NONE = iota // no version, pack all raw files into a zip package
	EPUB_VERSION_200         // epub version 2.0
	EPUB_VERSION_300         // epub version 3.0

	epub_NORMAL_FILE      = 1 << iota // nomal files
	epub_CONTENT_FILE                 // content files: the chapters
	epub_FULL_SCREEN_PAGE             // full screen pages in content
	epub_INTERNAL_FILE                // internal file, generated automatically in most case
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
	id          string
	name        string
	author      string
	publisher   string
	description string
	language    string
	cover       string // path of the cover image
	duokan      bool   // if duokan externsion is enabled
	files       []*File
}

func NewEpub(duokan bool) *Epub {
	this := new(Epub)
	this.files = make([]*File, 0, 256)
	this.duokan = duokan
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

func (this *Epub) Publisher() string {
	return this.publisher
}

func (this *Epub) SetPublisher(publisher string) {
	this.publisher = publisher
}

func (this *Epub) Description() string {
	return this.description
}

func (this *Epub) SetDescription(desc string) {
	this.description = desc
}

func (this *Epub) Language() string {
	return this.language
}

func (this *Epub) SetLanguage(lang string) {
	this.language = lang
}

func (this *Epub) Duokan() bool {
	return this.duokan
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
		path == path_of_nav_xhtml ||
		path == strings.ToLower(path_of_container_xml) {
		f.Attr = epub_INTERNAL_FILE
	}
	this.files = append(this.files, f)
}

func generateImagePage(path, alt string) []byte {
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
		"</html>\n", alt, path)
	return []byte(s)
}

func (this *Epub) AddFullScreenImage(path, alt string, chapters []Chapter) {
	f := &File{
		Path:     fmt.Sprintf("full_scrn_img_%04d.html", len(this.files)),
		Data:     generateImagePage(path, alt),
		Attr:     epub_CONTENT_FILE | epub_FULL_SCREEN_PAGE,
		Chapters: chapters,
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

func (this *Epub) generateContainerXml() []byte {
	return []byte("" +
		"<?xml version=\"1.0\"?>\n" +
		"<container version=\"1.0\" xmlns=\"urn:oasis:names:tc:opendocument:xmlns:container\">\n" +
		"	<rootfiles>\n" +
		"		<rootfile full-path=\"" + path_of_content_opf + "\" media-type=\"application/oebps-package+xml\"/>\n" +
		"	</rootfiles>\n" +
		"</container>")
}

func (this *Epub) generateContentOpf(version int) []byte {
	buf := new(bytes.Buffer)

	buf.WriteString("<?xml version='1.0' encoding='utf-8'?>\n")
	if version == EPUB_VERSION_200 {
		buf.WriteString("<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"2.0\" unique-identifier=\"uuid_id\">\n")
	} else {
		buf.WriteString("<package xmlns=\"http://www.idpf.org/2007/opf\" version=\"3.0\" unique-identifier=\"uuid_id\">\n")
	}
	buf.WriteString("	<metadata xmlns:opf=\"http://www.idpf.org/2007/opf\" xmlns:dc=\"http://purl.org/dc/elements/1.1/\">\n")

	fmt.Fprintf(buf, "		<dc:identifier id=\"uuid_id\">%s</dc:identifier>\n"+
		"		<dc:title>%s</dc:title>\n"+
		"		<dc:language>%s</dc:language>\n"+
		"		<meta name=\"cover\" content=\"%s\"/>\n",
		this.Id(),
		this.Name(),
		this.Language(),
		this.cover,
	)

	if version == EPUB_VERSION_200 {
		fmt.Fprintf(buf, "		<dc:creator opf:role=\"aut\">%s</dc:creator>\n", this.Author())
		fmt.Fprintf(buf, "		<dc:date>%s</dc:date>\n", time.Now().UTC().Format(time.RFC3339))
	} else {
		fmt.Fprintf(buf, "		<dc:creator id=\"creator\">%s</dc:creator>\n", this.Author())
		buf.WriteString("		<meta refines=\"#creator\" property=\"role\" scheme=\"marc:relators\" id=\"role\">aut</meta>\n")
		fmt.Fprintf(buf, "		<meta property=\"dcterms:modified\">%s</meta>\n", time.Now().UTC().Format(time.RFC3339))
	}

	if len(this.Publisher()) > 0 {
		fmt.Fprintf(buf, "<dc:publisher>%s</dc:publisher>\n", this.Publisher())
	}

	if len(this.Description()) > 0 {
		fmt.Fprintf(buf, "<dc:description>%s</dc:description>\n", this.Description())
	}

	buf.WriteString("	</metadata>\n	<manifest>\n")

	if version == EPUB_VERSION_200 {
		buf.WriteString("		<item id=\"ncx\" href=\"" + path_of_toc_ncx + "\" media-type=\"application/x-dtbncx+xml\"/>\n")
	} else {
		buf.WriteString("		<item properties=\"nav\" id=\"ncx\" href=\"" + path_of_nav_xhtml + "\" media-type=\"application/xhtml+xml\"/>\n")
	}

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

	if version == EPUB_VERSION_200 {
		buf.WriteString("	</manifest>\n	<spine toc=\"ncx\">\n")
	} else {
		buf.WriteString("	</manifest>\n	<spine>\n")
	}

	if len(this.cover) > 0 {
		buf.WriteString("		<itemref idref=\"cover\" linear=\"no\"")
		if this.duokan {
			buf.WriteString(" properties=\"duokan-page-fullscreen\"/>\n")
		} else {
			buf.WriteString("/>\n")
		}
	}

	for i, f := range this.files {
		if (f.Attr & epub_CONTENT_FILE) == 0 {
			continue
		}
		fmt.Fprintf(buf, "		<itemref idref=\"item%04d\" linear=\"yes\"", i)
		if this.duokan && (f.Attr&epub_FULL_SCREEN_PAGE) != 0 {
			buf.WriteString(" properties=\"duokan-page-fullscreen\"/>\n")
		} else {
			buf.WriteString("/>\n")
		}
	}

	buf.WriteString("	</spine>\n</package>")

	return buf.Bytes()
}

////////////////////////////////////////////////////////////////////////////////
// epub 2.0

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
		"	<docAuthor><text>%s</text></docAuthor>\n"+
		"	<navMap>\n",
		this.Id(),
		this.Depth(),
		this.Name(),
		this.Author(),
	)

	depth, playorder := 0, 0
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

	buf.WriteString("	</navMap>\n</ncx>")

	return buf.Bytes()
}

////////////////////////////////////////////////////////////////////////////////
// epub 3.0

func (this *Epub) generateNavXhtml() []byte {
	buf := new(bytes.Buffer)
	fmt.Fprintf(buf,
		"<?xml version=\"1.0\" encoding=\"utf-8\"?>\n"+
			"<html xmlns=\"http://www.w3.org/1999/xhtml\" xmlns:epub=\"http://www.idpf.org/2007/ops\">\n"+
			"	<head>\n"+
			"		<title>%s</title>\n"+
			"	</head>\n"+
			"	<body>\n"+
			"		<nav id=\"toc\" epub:type=\"toc\">\n",
		this.Name(),
	)

	depth, playorder := 0, 0
	for _, f := range this.files {
		if (f.Attr & epub_CONTENT_FILE) == 0 {
			continue
		}
		for _, c := range f.Chapters {
			if c.Level == depth {
				buf.WriteString("</li>\n<li")
			} else if c.Level > depth {
				buf.WriteString("<ol>\n<li")
				depth = c.Level
			} else {
				for c.Level < depth {
					buf.WriteString("</li>\n</ol>\n")
					depth--
				}
				buf.WriteString("</li>\n<li")
			}
			fmt.Fprintf(buf,
				" id=\"chapter_%d\">\n	<a href=\"%s\">%s</a>\n",
				playorder,
				f.Path+c.Link,
				c.Title,
			)
			playorder++
		}
	}

	for depth > 0 {
		buf.WriteString("</li>\n</ol>\n")
		depth--
	}

	buf.WriteString("		</nav>\n	</body>\n</html>")

	return buf.Bytes()
}

////////////////////////////////////////////////////////////////////////////////

func (this *Epub) Build(version int) ([]byte, error) {
	compressor := epubCompressor{}
	if e := compressor.init(); e != nil {
		return nil, e
	}

	if version != EPUB_VERSION_NONE {
		data := this.generateContainerXml()
		if e := compressor.addFile(path_of_container_xml, data); e != nil {
			return nil, e
		}
		data = this.generateContentOpf(version)
		if e := compressor.addFile(path_of_content_opf, data); e != nil {
			return nil, e
		}
		if version == EPUB_VERSION_200 {
			data = this.generateTocNcx()
			if e := compressor.addFile(path_of_toc_ncx, data); e != nil {
				return nil, e
			}
		} else {
			data = this.generateNavXhtml()
			if e := compressor.addFile(path_of_nav_xhtml, data); e != nil {
				return nil, e
			}
		}
		if len(this.cover) > 0 {
			data = generateImagePage(this.cover, "cover")
			if e := compressor.addFile(path_of_cover_page, data); e != nil {
				return nil, e
			}
		}
	}

	for _, f := range this.files {
		if e := compressor.addFile(f.Path, f.Data); e != nil {
			return nil, e
		}
	}

	if e := compressor.close(); e != nil {
		return nil, e
	}

	return compressor.result(), nil
}

func (this *Epub) Save(path string, version int) error {
	data, e := this.Build(version)
	if e != nil {
		return e
	}

	f, e := os.Create(path)
	if e == nil {
		_, e = f.Write(data)
		f.Close()
	}

	return e
}
