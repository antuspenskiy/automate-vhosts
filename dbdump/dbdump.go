package main

import (
	"os/exec"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"flag"
	"time"
	"os"
	"fmt"

	"github.com/antuspenskiy/automate-vhosts/branch"
)

// Set the command line arguments
var (
	mysqlUser     = flag.String("u", "test", "Name of your database user.")
	mysqlHost     = flag.String("h", "localhost", "Name of your Mysql hostname.")
	mysqlDb       = flag.String("db", "test", "Database name.")
	allDatabase   = flag.Bool("db-all", false, "If set dump all Mysql databases.")
	backupDir     = flag.String("backup-dir", "/opt/backup/db", "Backup directory for dumps.")
	storageDir    = flag.String("storage-dir", "/mnt/backup", "Remote storage directory for dumps.")
	gzipEnable    = flag.Bool("gzip", true, "If set gzip compression enabled.")
)

func main() {

	// Get command line arguments
	flag.Parse()
	flag.Args()

	// Get the hostname
	hostname, err := os.Hostname()

	filename := ""
	current := time.Now()
	now := fmt.Sprintf(current.Format("20060102.150405"))

	// Set Filename
	if *allDatabase {
		fmt.Printf("Output: Dumping %s databases's start...\n", hostname)
		filename = fmt.Sprintf("%s_%s.sql", hostname, now)
	} else {
		fmt.Printf("Output: Dumping database %s start...\n", *mysqlDb)
		filename = fmt.Sprintf("%s_%s.sql", *mysqlDb, now)
	}

	if *gzipEnable {
		filename += ".gz"
	}

	// Define local tmp file
	localTmpFile := fmt.Sprintf("%s/%s", *backupDir, filename)

	// Compose mysqldump command
	mysqldumpCommand := fmt.Sprintf("mysqldump -u%s -h%s --single-transaction ", *mysqlUser, *mysqlHost)
	if *allDatabase {
		mysqldumpCommand += "--all-databases "
	} else if *mysqlDb != "" {
		mysqldumpCommand += *mysqlDb
	} else {
		fmt.Println("You must specify a database name")
	}

	// TODO: Refactor to ExecCmd fun or similar
	// Create database dump and store it on local tmp file
	cmd := exec.Command("/bin/bash", "-c", mysqldumpCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Create a gzip file of the dbdump output stream
	if *gzipEnable {
		var outGzip bytes.Buffer
		w := gzip.NewWriter(&outGzip)
		w.Write(out.Bytes())
		w.Close()

		out = outGzip
	}

	// Write the gzip stream to a tmp file
	ioutil.WriteFile(localTmpFile, out.Bytes(), 0666)
	fmt.Printf("Output: Gzip file %s created\n", localTmpFile)

	// TODO: Better to use semicolon for rm {} \;
	// Rotate dumps then synchronize it via rsync
	branch.ExecCmd("bash", "-c", fmt.Sprintf("find %s/ -name '*.sql.gz' -type f -mtime +14 -exec rm {} +", *backupDir))

	// Synchronize backup directory with storage directory
	branch.ExecCmd("bash", "-c", fmt.Sprintf("rsync -avpze --progress --stats --delete %s/ %s/", *backupDir, *storageDir))

	fmt.Printf("Output: Dump database %s finished.\n", *mysqlDb)
}
