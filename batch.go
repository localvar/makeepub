package main

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

type taskResult struct {
	input string
	e     error
}

var (
	chTaskResult chan *taskResult
)

func runTask(input string, outdir string) {
	tr := new(taskResult)
	tr.input = input

	em := NewEpubMaker(nil)
	if tr.e = em.RunPhisical(input); tr.e == nil {
		tr.e = em.SaveTo(outdir)
	}

	chTaskResult <- tr
}

func processBatchFile(f *os.File, outdir string) (count int, e error) {
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		n := strings.TrimSpace(scanner.Text())
		if len(n) == 0 {
			continue
		}

		go runTask(n, outdir)
		count++
	}

	if e = scanner.Err(); e != nil {
		logger.Println("error reading batch file.")
	}

	return
}

func processBatchFolder(f *os.File, outdir string) (int, error) {
	names, e := f.Readdirnames(-1)
	if e != nil {
		logger.Println("error reading source folder.")
		return 0, e
	}

	count := 0
	for _, name := range names {
		name = filepath.Join(f.Name(), name)

		stat, e := os.Stat(name)
		// let runTask to handle the error if 'e' is not nil
		if e == nil && (!stat.IsDir()) {
			if strings.ToLower(filepath.Ext(name)) != ".zip" {
				continue
			}
		}

		go runTask(name, outdir)
		count++
	}

	return count, nil
}

func RunBatch() {
	checkCommandLine(3)

	f, e := os.Open(os.Args[2])
	if e != nil {
		logger.Fatalf("failed to open '%s'.\n", os.Args[2])
	}
	defer f.Close()

	outdir := ""
	if len(os.Args) > 3 {
		outdir = os.Args[3]
	}

	runtime.GOMAXPROCS(runtime.NumCPU() + 1)
	chTaskResult = make(chan *taskResult)
	defer close(chTaskResult)

	var count int
	if fi, _ := f.Stat(); fi.IsDir() {
		count, e = processBatchFolder(f, outdir)
	} else {
		count, e = processBatchFile(f, outdir)
	}

	if e != nil && count == 0 {
		return
	}

	failed := 0
	for i := 0; i < count; i++ {
		if (<-chTaskResult).e != nil {
			failed++
		}
	}

	logger.Printf("total: %d   succeeded: %d    failed: %d\n", count, count-failed, failed)
}
