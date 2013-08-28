package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func showUsage() {
	usage := `Create/Batch Create/Pack/Extract EPUB file(s). Merge HTML/Text files.
It can also work as a web server to convert an uploaded zip file to an EPUB.
Please refer to manual for detailed usage.

COMMAND LINE
  Create       : makeepub <VirtualFolder> [OutputFolder]
  Batch Create : makeepub -b <InputFolder> [OutputFolder]
                 makeepub -b <BatchFile> [OutputFolder]
  Pack         : makeepub -p <VirtualFolder> <OutputFile>
  Extract      : makeepub -e <EpubFile> <OutputFolder>
  Merge HTML   : makeepub -mh <VirtualFolder> <OutputFile>
  Merge Text   : makeepub -mt <VirtualFolder> <OutputFile>
  Web Server   : makeepub -s [Port]

ARGUMENT
  VirtualFolder: An OS folder or a zip file which contains the input files.
  OutputFolder : An OS folder to store the output file(s).
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

func CheckCommandLineArgumentCount(minArg int) {
	if len(os.Args) < minArg {
		onCommandLineError()
	}
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
	if cmd[0] != '-' && cmd[0] != '/' {
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

func removeUtf8Bom(data []byte) []byte {
	if len(data) > 3 && data[0] == 0xEF && data[1] == 0xBB && data[2] == 0xBF {
		data = data[3:]
	}
	return data
}

func main() {
	logger.Println("project home page: https://github.com/localvar/makeepub")
	CheckCommandLineArgumentCount(2)

	AddCommandHandler("?", showUsage)
	AddCommandHandler("h", showUsage)
	handler := findCommandHandler(os.Args[1])

	start := time.Now()
	handler()
	logger.Println("done, time used:", time.Now().Sub(start).String())

	os.Exit(0)
}
