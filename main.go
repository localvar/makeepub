package main

import (
	"fmt"
	"log"
	"os"
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
	fmt.Printf("Usage: %s <folder>  [output]\n", bn)
	fmt.Printf("       %s <zip> [output]\n", bn)
	fmt.Printf("       %s <-? | -h | -H>\n", bn)
	os.Exit(0)
}

func checkCommandLine(minArg int) {
	if len(os.Args) < minArg {
		log.Fatalf("invalid command line. see '%s -?'\n", getBinaryName())
	}
}

func main() {
	log.SetFlags(0)
	log.SetPrefix(getBinaryName() + ": ")

	checkCommandLine(2)

	start := time.Now()

	switch os.Args[1] {
	case "-b", "-B", "/b", "/B":
		RunBatch()
	case "-e", "-E", "/e", "/E":
		RunExtract()
	case "-h", "-H", "/h", "/H", "-?":
		showUsage()
	default:
		RunMake()
	}

	log.Println("done, time used:", time.Now().Sub(start).String())
	os.Exit(0)
}
