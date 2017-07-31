package main

import (
	"fmt"
	"os"
	"time"
	"os/exec"
	"strings"

	"github.com/antuspenskiy/automate-vhosts/branch"
)

func main() {

	c := branch.LoadConfig("../config/config.json")

	fmt.Printf("%+v\n\n", c)

	// Main variables
	dbName := fmt.Sprintf("i_%s", branch.PassArguments())
	current := time.Now()
	dumpFileFormat := fmt.Sprintf(current.Format("20060102.150405"))
	dumpFileDst := fmt.Sprintf("%s/dump_%s.sql", c.DatabaseDir, dumpFileFormat)

	if branch.PathExist(c.StorageDir) {
		os.Chdir(c.StorageDir)
	} else {
		fmt.Printf("Error: No such file or directory %v\n", c.StorageDir)
		os.Exit(1)
	}

	// Pipeline commands
	ls := exec.Command("find", ".", "-name", "*.sql.gz")
	tail := exec.Command("tail", "-1")

	// Run the pipeline
	output, stderr, err := branch.PipeLine(ls, tail)

	if err != nil {
		fmt.Printf("Error: %s", err)
	}

	// Print the stdout, if any
	//if len(output) > 0 {
	//	fmt.Printf("Output: %s\n", output)
	//}

	// Print the stderr, if any
	if len(stderr) > 0 {
		fmt.Printf("%q: (stderr)\n", stderr)
	}

	// Convert byte output to string, trim it and use in cmd
	dumpFileStr := string(output[:])
	dumpFileSrc := strings.TrimSpace(dumpFileStr)

	// Copy last database dump to dbDir
	branch.ExecCmd("rsync", "-P", "-t", dumpFileSrc, c.DatabaseDir)

	os.Chdir(c.StorageDir)

	// Extract database dump, use time() for each extracted file *.sql
	branch.UnpackGzipFile(dumpFileSrc, dumpFileDst)

	// Prepare database
	branch.ExecCmd("mysql", "-u", "test", "-e", fmt.Sprintf("DROP DATABASE IF EXISTS %s; "+
		"CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci; "+
		"GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';", dbName, dbName, dbName, dbName, dbName))

	// Import database dump
	// TODO: Use password
	branch.ExecCmd("bash", "-c", fmt.Sprintf("mysql -utest %s < %s", dbName, dumpFileDst))

	// Delete database dump's
	branch.DeleteFile(dumpFileSrc)
	branch.DeleteFile(dumpFileDst)
}
