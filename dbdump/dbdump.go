package main

import (
	"os/exec"
	"bytes"
	"compress/gzip"
	"io/ioutil"
	"log"
	"flag"
	"time"
	"os"
	"fmt"
)

// Set the command line arguments
var (
	mysqlUser   = flag.String("db-user", "test", "Name of your MySql dbdump USER")
	mysqlHost   = flag.String("db-host", "localhost", "Name of your MySql dbdump HOST")
	mysqlDb     = flag.String("db", "i_66chucknorris", "Database Name")
	allDatabase = flag.Bool("dbdump-all", false, "If set script dbdump all MySql Databases")
	tmpDir      = flag.String("tmp-dir", "/tmp", "Temp directory (default /tmp)")
	logDir      = flag.String("log-dir", "/tmp", "Log directory (default /var/log)")
	gzipEnable  = flag.Bool("gzip", true, "If set Gzip Compression Enabled")
)

func main() {

	// Get command line arguments
	flag.Parse()

	// Get the hostname
	hostname, err := os.Hostname()

	filename := ""
	current := time.Now()
	now := fmt.Sprintf(current.Format("20060102.150405"))

	// Open the log file
	ol, err := os.OpenFile(*logDir+"/go-mysql-dbdump.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer ol.Close()

	log.SetOutput(ol)

	// Set Filename
	if *allDatabase {
		log.Println("Dump " + hostname + " DBs Start")
		filename = hostname + "_" + now + ".sql"
	} else {
		log.Println("Dump DB " + *mysqlDb + " Start")
		filename = *mysqlDb + "_" + now + ".sql"
	}

	if *gzipEnable {
		filename += ".gz"
	}

	// Define local tmp file
	localTmpFile := *tmpDir + "/" + filename

	// Compose mysqldump command
	mysqldumpCommand := "mysqldump --single-transaction -u " + *mysqlUser + " -h " + *mysqlHost + " "
	if *allDatabase {
		mysqldumpCommand += "--all-databases "
	} else if *mysqlDb != "" {
		mysqldumpCommand += *mysqlDb
	} else {
		log.Fatal("You must specify a DB Name")
	}

	// Create database dbdump and store it on local tmp file
	cmd := exec.Command("/bin/bash", "-c", mysqldumpCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		log.Fatal("Mysqldump Error", err)
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

	log.Println("Dump DB " + *mysqlDb + " Finish")
}
