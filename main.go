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

func checkCommandLineArgumentCount(minArg int) {
	if len(os.Args) < minArg {
		onCommandLineError()
	}
}

var logger = log.New(os.Stderr, "makeepub: ", 0)

func main() {
	checkCommandLineArgumentCount(2)

	start := time.Now()

	mode := strings.ToLower(os.Args[1])
	if mode[0] != '-' && mode[0] != '/' {
		RunMake()
	} else {
		switch mode[1:] {
		case "b":
			RunBatch()
		case "e":
			RunExtract()
		case "h", "?":
			showUsage()
		case "mh", "mt":
			RunMerge()
		case "p":
			RunPack()
		case "s":
			RunServer()
		default:
			onCommandLineError()
		}
	}

	logger.Println("done, time used:", time.Now().Sub(start).String())
	os.Exit(0)
}
