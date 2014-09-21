package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

const version = "1.1.0"

func showUsage() {
	usage := `Create/Batch Create/Pack/Extract EPUB file(s). Merge HTML/Text files.
It can also work as a web server to convert an uploaded zip file to an EPUB.
Please refer to manual for detailed usage.

COMMAND LINE
  Create       : makeepub <VirtualFolder> [OutputFolder] [-epub2] [-noduokan]
  Batch Create : makeepub -b <InputFolder> [OutputFolder] [-epub2] [-noduokan]
                 makeepub -b <BatchFile> [OutputFolder] [-epub2] [-noduokan]
  Pack         : makeepub -p <VirtualFolder> <OutputFile>
  Extract      : makeepub -e <EpubFile> <OutputFolder>
  Merge HTML   : makeepub -mh <VirtualFolder> <OutputFile>
  Merge Text   : makeepub -mt <VirtualFolder> <OutputFile>
  Web Server   : makeepub -s [Port]

ARGUMENT
  VirtualFolder: An OS folder or a zip file which contains the input files.
  OutputFolder : An OS folder to store the output file(s).
  -epub2       : Generate books using EPUB2 format, otherwise EPUB3.
  -noduokan    : Disable DuoKan externsion.
  InputFolder  : An OS folder which contains the input folder(s)/file(s).
  BatchFile    : A text which lists the path of 'VirtualFolders' to be
                 processed, one line for one 'VirtualFolder'
  OutputFile   : The path of the output file.
  EpubFile     : The path of an EPUB file.
  Port         : The TCP port to listen to, default value is 80.
`
	fmt.Print(usage)
	os.Exit(0)
}

func onCommandLineError() {
	logger.Fatalln("invalid command line. see 'makeepub -?'")
}

func getArg(index int, dflt string) string {
	count := 0
	for _, arg := range os.Args[1:] {
		if !isFlag(arg) {
			if count == index {
				return arg
			}
			count++
		}
	}
	return dflt
}

func getFlag(index int) string {
	count := 0
	for _, arg := range os.Args[1:] {
		if isFlag(arg) {
			if count == index {
				return arg[1:]
			}
			count++
		}
	}
	return ""
}

func isFlag(arg string) bool {
	if os.PathSeparator == '/' {
		return arg[0] == '-'
	}
	return arg[0] == '-' || arg[0] == '/'
}

func getFlagBool(flag string) bool {
	flag = strings.ToLower(flag)
	for _, arg := range os.Args[1:] {
		if isFlag(arg) && strings.ToLower(arg[1:]) == flag {
			return true
		}
	}
	return false
}

type CommandHandler struct {
	command string
	handler func()
}

var (
	logger   = log.New(os.Stderr, "makeepub: ", 0)
	handlers = make([]CommandHandler, 0, 8)
)

func AddCommandHandler(cmd string, handler func()) {
	for _, h := range handlers {
		if h.command == cmd {
			logger.Fatalf("handler for command '%s' already exists.\n", cmd)
		}
	}
	handlers = append(handlers, CommandHandler{command: cmd, handler: handler})
}

func findCommandHandler(cmd string) func() {
	if !isFlag(cmd) {
		return RunMake
	}
	cmd = strings.ToLower(cmd[1:])
	for _, h := range handlers {
		if cmd == h.command {
			return h.handler
		}
	}
	return onCommandLineError
}

func main() {
	fmt.Println("makeepub v" + version + ", home page: https://github.com/localvar/makeepub")
	if len(os.Args) < 2 {
		onCommandLineError()
	}

	AddCommandHandler("?", showUsage)
	AddCommandHandler("h", showUsage)
	handler := findCommandHandler(os.Args[1])

	start := time.Now()
	handler()
	logger.Println("done, time used:", time.Now().Sub(start).String())

	os.Exit(0)
}
