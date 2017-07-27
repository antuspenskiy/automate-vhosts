package main

import (
	"fmt"
	"os"
	"time"
	"flag"
	"strings"
	"os/exec"
	"log"

	"github.com/antuspenskiy/automate-vhosts/branch"
)

func main() {
	flag.Parse()

	branchName := flag.Args()
	if len(branchName) == 1 {
		log.Fatal("Error: No files specified.")
	}
	fmt.Println("Output: Using file:", branchName)

	// Main variables
	dbDir := "/tmp"
	storageDir := "/Users/auspenskii/test"
	current := time.Now()
	dumpFilename := fmt.Sprintf(current.Format("20060102.150405"))
	dumpPath := fmt.Sprintf("%s/dump_%s.sql", dbDir, dumpFilename)
	fmt.Println(dumpPath)

	// Check if storageDir is exist
	branch.IsExist(storageDir)
	os.Chdir(storageDir)

	// Pipeline commands
	ls := exec.Command("find", ".", "-name", "*.sql.gz")
	tail := exec.Command("tail", "-1")

	// Run the pipeline
	output, stderr, err := branch.PipeLine(ls, tail)

	if err != nil {
		fmt.Printf("Error: %s", err)
	}

	// Print the stdout, if any
	if len(output) > 0 {
		fmt.Printf("Output: %s", output)
	}

	// Print the stderr, if any
	if len(stderr) > 0 {
		fmt.Printf("%q: (stderr)", stderr)
	}

	// Convert byte output to string, trim it and use in cmd
	outputStr := string(output[:])
	outputTrim := strings.TrimSpace(outputStr)

	// Copy last database dump to dbDir
	branch.ExecCmd("rsync", "-P", "-t", outputTrim, dbDir)

	os.Chdir(dbDir)

	// Extract database dump
	branch.UnpackGzipFile(outputTrim, dumpPath)

}
