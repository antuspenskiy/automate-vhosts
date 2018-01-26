package main

import (
	"bytes"
	"compress/gzip"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os/exec"
	"time"

	"github.com/antuspenskiy/automate-vhosts/pkg/branch"
)

var (
	// VERSION used to show version of CLI
	VERSION = "undefined"
	// BUILDTIME used to show buildtime of CLI
	BUILDTIME = "undefined"
	// COMMIT used to show commit when CLI compiled
	COMMIT = "undefined"
	// BRANCH used to show branchname when CLI compiled
	BRANCH = "undefined"
)

func main() {
	fmt.Printf("Version    : %s\n", VERSION)
	fmt.Printf("Git Hash   : %s\n", COMMIT)
	fmt.Printf("Build Time : %s\n", BUILDTIME)
	fmt.Printf("Branch     : %s\n\n", BRANCH)

	// Set the command line arguments
	var (
		mysqlUser   = flag.String("u", "test", "Name of your database user.")
		mysqlHost   = flag.String("h", "localhost", "Name of your Mysql hostname.")
		mysqlDb     = flag.String("db", "test", "Database name.")
		allDatabase = flag.Bool("db-all", false, "If set dump all Mysql databases.")
		backupDir   = flag.String("backup-dir", "/opt/backup/db", "Backup directory for dumps.")
		storageDir  = flag.String("storage-dir", "/mnt/backup", "Remote storage directory for dumps.")
		gzipEnable  = flag.Bool("gzip", true, "If set gzip compression enabled.")
		filename    string
	)

	// Get command line arguments
	flag.Parse()

	// Get server hostname
	hostname := branch.GetHostname()

	current := time.Now()
	now := current.Format("20060102.150405")

	// Set Filename
	if *allDatabase {
		log.Printf("Dumping %s databases's start...\n", hostname)
		filename = fmt.Sprintf("%s_%s.sql", hostname, now)
	} else {
		log.Printf("Dumping database %s start...\n", *mysqlDb)
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
		log.Println("You must specify a database name")
	}

	// TODO: Refactor to ExecCmd func or similar
	// Create database dump and store it on local tmp file
	cmd := exec.Command("/bin/bash", "-c", mysqldumpCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Error: %v\n", err)
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
	log.Printf("Gzip file %s created\n", localTmpFile)

	// TODO: Better to use semicolon for rm {} \;
	// Rotate dumps then synchronize it via rsync
	branch.RunCommand("bash", "-c", fmt.Sprintf("find %s/ -name '*.sql.gz' -type f -mtime +14 -exec rm {} +", *backupDir))

	// Synchronize backup directory with storage directory
	branch.RunCommand("bash", "-c", fmt.Sprintf("rsync -avpze --progress --stats --delete %s/ %s/", *backupDir, *storageDir))

	log.Printf("Dump database %s finished.\n", *mysqlDb)

}
