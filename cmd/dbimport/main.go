package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/antuspenskiy/automate-vhosts/pkg/branch"
	_ "github.com/go-sql-driver/mysql"
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
		refSlug       = flag.String("refslug", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		mysqlUser     = flag.String("user", "", "Name of your database user.")
		mysqlPassword = flag.String("password", "", "Name of your database user password.")
		mysqlHostname = flag.String("hostname", "localhost", "Name of your database hostname.")
		mysqlPort     = flag.String("port", "3306", "Name of your database port.")
		mysqlDatabase = flag.String("database", "", "Name of your database.")
	)

	// Get command line arguments
	flag.Parse()

	// Load json configuration
	conf, err := branch.ReadConfig("env")
	branch.Check(err)

	// Get server hostname
	hostname := branch.GetHostname()

	// Main variables
	dbName := branch.ParseBranchName(*refSlug)

	// Use Format for extracted file, so they don't conflicted
	current := time.Now()
	dumpFileDst := (fmt.Sprintf("%s/dump_%s.sql", conf.GetString("dbdir"), (current.Format("20060102.150405"))))

	if branch.DirectoryExists(conf.GetString("storagedir")) {
		err = os.Chdir(conf.GetString("storagedir"))
		branch.Check(err)

		// Pipeline commands
		dumpFile, err := exec.Command("/bin/bash", "-c", "find . -name '*.sql.gz' | tail -1").CombinedOutput()
		branch.Check(err)
		fmt.Printf("\nGet latest database dump:\n\n%s\n", dumpFile)

		// Convert byte output to string
		dumpFileStr := string(dumpFile[:])
		dumpFileSrc := strings.TrimSpace(dumpFileStr)

		// Copy last database dbdump to dbDir
		branch.RunCommand("rsync", "-P", "-t", dumpFileSrc, conf.GetString("dbdir"))

		err = os.Chdir(conf.GetString("dbdir"))
		branch.Check(err)

		// Extract database dbdump, use time() for each extracted file *.sql
		branch.Gunzip(dumpFileSrc, dumpFileDst)

		// Prepare database
		// [user[:pass]@][protocol[(addr)]]/dbname[?p1=v1&...]
		mysqlInfo := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			*mysqlUser, *mysqlPassword, *mysqlHostname, *mysqlPort, *mysqlDatabase)

		db, err := sql.Open("mysql", mysqlInfo)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer func() {
			err = db.Close()
			branch.Check(err)
		}()

		// make sure connection is available
		err = db.Ping()
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			log.Println("Successfully connected to MySQL!")
		}

		numdrop, err := branch.DropDB(db, dbName)
		branch.Check(err)
		log.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numdrop)

		numcreate, err := branch.CreateDB(db, dbName)
		branch.Check(err)
		log.Printf("MySQL: Running: CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numcreate)

		numgrant, err := branch.GrantUserPriv(db, dbName)
		branch.Check(err)
		log.Printf("MySQL: Running: GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';\n", dbName, dbName, dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numgrant)

		numflush, err := branch.FlushPriv(db)
		branch.Check(err)
		log.Printf("MySQL: Running: FLUSH PRIVILEGES;")
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numflush)

		// Import database dump
		branch.RunCommand("bash", "-c", fmt.Sprintf("time mysql -u%s %s < %s", *mysqlUser, dbName, dumpFileDst))

		if strings.Contains(hostname, "ees") {
			numsal, err := branch.DropSalary(db, dbName)
			branch.Check(err)
			log.Printf("MySQL: Running: UPDATE %s.user_data SET salary = 10000, salary_proposed = 11000;\n", dbName)
			log.Printf("MySQL: Query OK, %d rows affected\n\n", numsal)
		}
	} else {
		log.Fatalf("Error: No such file or directory %v\n", conf.GetString("storagedir"))
	}
	// Delete database dbdump's
	branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/*.sql*", conf.GetString("dbdir")))
}
