package main

import (
	"bytes"
	"fmt"
	"html"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

const (
	homePage = `<!DOCTYPE html>
<html>
	<head>
		<meta charset='utf-8' />
		<title>MakeEpub</title>
	</head>
	<body>
		<form enctype="multipart/form-data" action="/" method="POST">
    		<label>Source File / 源文件:</label><input name="input" type="file" />
    		<input type="submit" value="Upload & Make / 上传并转换">
		</form>
	</body>
</html>
`
	errorPage = `<!DOCTYPE html>
<html>
	<head>
		<meta charset='utf-8' />
		<title>MakeEpub</title>
	</head>
	<body>
		<h1>Failed to convert / 转换失败</h1>
		<p>%s</p>
		<p>%s</p>
	</body>
</html>
`
)

func doConvert(l *log.Logger, w http.ResponseWriter, r *http.Request) error {
	in, _, e := r.FormFile("input")
	if e != nil {
		return e
	}
	data, e := ioutil.ReadAll(in)
	in.Close()
	if e != nil {
		return e
	}

	folder, e := NewZipFolder(data)
	if e != nil {
		return e
	}

	maker := NewEpubMaker(l)
	if e = maker.Process(folder); e != nil {
		return e
	}

	data, name := maker.GetResult()
	w.Header().Add("Content-Disposition", "attachment; filename="+name)
	http.ServeContent(w, r, name, time.Now(), bytes.NewReader(data))

	return nil
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	buf := new(bytes.Buffer)
	l := log.New(buf, "", 0)

	if e := doConvert(l, w, r); e != nil {
		fmt.Fprintf(w,
			errorPage,
			html.EscapeString(buf.String()),
			html.EscapeString(e.Error()))
	}
}

func handler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		fmt.Fprint(w, homePage)
	} else if r.Method == "POST" {
		handleConvert(w, r)
	}
}

func RunServer() {
	port := 80
	if len(os.Args) > 2 {
		port += 0
		port, e := strconv.Atoi(os.Args[2])
		if e != nil || port <= 0 || port > 65535 {
			logger.Fatalln("invalid port number.")
		}
	}
	fmt.Printf("Web Server started, listen at port '%d'\n", port)
	fmt.Println("Press 'Ctrl + C' to exit.")
	http.HandleFunc("/", handler)
	http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
}
