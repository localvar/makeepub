package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

func getBinaryName() string {
	name := os.Args[0]
	for i := len(name) - 1; i >= 0; i-- {
		if name[i] == '.' {
			name = name[0:i]
		} else if os.IsPathSeparator(name[i]) {
			name = name[i+1:]
			break
		}
	}
	return name
}

func showUsage() {
	bn := getBinaryName()
	fmt.Printf("Usage: %s <input_folder>  [output_folder]\n", bn)
	fmt.Printf("       %s <zip> [output_folder]\n", bn)
	fmt.Printf("       %s <epub> <output_folder>\n", bn)
	fmt.Printf("       %s <input_folder> <output_file>", bn)
	fmt.Printf("       %s <-? | -h | -H>\n", bn)
	os.Exit(0)
}

func checkCommandLine(minArg int) {
	if len(os.Args) < minArg {
		logger.Fatalf("invalid command line. see '%s -?'\n", getBinaryName())
	}
}

var logger *log.Logger

func main() {
	logger = log.New(os.Stderr, getBinaryName()+": ", 0)

	checkCommandLine(2)

	start := time.Now()

	switch strings.ToLower(os.Args[1]) {
	case "-b", "/b":
		RunBatch()
	case "-e", "/e":
		RunExtract()
	case "-h", "/h", "-?":
		showUsage()
	case "-mh", "/mh", "-mt", "/mt":
		RunMerge()
	case "-p", "/p":
		RunPack()
	default:
		RunMake()
	}

	logger.Println("done, time used:", time.Now().Sub(start).String())
	os.Exit(0)
}
