package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/antuspenskiy/automate-vhosts/pkg/archive"
	"github.com/antuspenskiy/automate-vhosts/pkg/cmd"
	"github.com/antuspenskiy/automate-vhosts/pkg/config"
	"github.com/antuspenskiy/automate-vhosts/pkg/db"
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
	conf, err := config.ReadConfig("env")
	cmd.Check(err)

	// Get server hostname
	hostname := cmd.GetHostname()

	// Main variables
	dbName := db.ParseBranchName(*refSlug)

	// Use Format for extracted file, so they don't conflicted
	current := time.Now()
	tarExtractFile := fmt.Sprintf("dump_%s.sql", current.Format("20060102.150405"))
	tarExtractDst := path.Join(conf.GetString("dbdir"), tarExtractFile)

	// Get latest dump from storage dir
	files, err := cmd.FilePathWalkDir(conf.GetString("storagedir"))
	cmd.Check(err)

	fname := make([]string, 0)
	cmd.Check(err)

	for _, file := range files {
		fname = append(fname, file)
	}
	// Sorting arrays
	sort.Slice(fname, func(i, j int) bool {
		return fname[i] < fname[j]
	})
	// Get last dump file
	tarFile := fname[len(fname)-1]

	err = os.Chdir(conf.GetString("storagedir"))
	cmd.Check(err)

	// Copy last database dump to local directory
	cmd.RunCommand("rsync", "-P", "-t", tarFile, conf.GetString("dbdir"))

	err = os.Chdir(conf.GetString("dbdir"))
	cmd.Check(err)

	// Extract *.tar.gz archive
	archive.ExtractTarGz(tarFile, tarExtractDst)

	// Prepare database
	// [user[:pass]@][protocol[(addr)]]/dbname[?p1=v1&...]
	mysqlInfo := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
		*mysqlUser, *mysqlPassword, *mysqlHostname, *mysqlPort, *mysqlDatabase)

	conn, err := sql.Open("mysql", mysqlInfo)
	if err != nil {
		fmt.Println(err.Error())
	}
	defer func() {
		err = conn.Close()
		cmd.Check(err)
	}()

	// make sure connection is available
	err = conn.Ping()
	if err != nil {
		log.Fatalf(err.Error())
	} else {
		log.Println("Successfully connected to MySQL!")
	}

	numdrop, err := db.DropDB(conn, dbName)
	cmd.Check(err)
	log.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
	log.Printf("MySQL: Query OK, %d rows affected\n\n", numdrop)

	numcreate, err := db.CreateDB(conn, dbName)
	cmd.Check(err)
	log.Printf("MySQL: Running: CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;\n", dbName)
	log.Printf("MySQL: Query OK, %d rows affected\n\n", numcreate)

	numgrant, err := db.GrantUserPriv(conn, dbName)
	cmd.Check(err)
	log.Printf("MySQL: Running: GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';\n", dbName, dbName, dbName)
	log.Printf("MySQL: Query OK, %d rows affected\n\n", numgrant)

	numflush, err := db.FlushPriv(conn)
	cmd.Check(err)
	log.Printf("MySQL: Running: FLUSH PRIVILEGES;")
	log.Printf("MySQL: Query OK, %d rows affected\n\n", numflush)

	// Import database dump
	cmd.RunCommand("bash", "-c", fmt.Sprintf("time mysql -u%s %s < %s", *mysqlUser, dbName, tarExtractDst))

	if strings.Contains(hostname, "ees") {
		numsal, err := db.DropSalary(conn, dbName)
		cmd.Check(err)
		log.Printf("MySQL: Running: UPDATE %s.user_data SET salary = 10000, salary_proposed = 11000;\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numsal)
	}
	// Delete database dbdump's
	cmd.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/*.sql*", conf.GetString("dbdir")))
}
