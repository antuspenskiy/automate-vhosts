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
)

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	VERSION   = "undefined"
	BUILDTIME = "undefined"
	COMMIT    = "undefined"
	BRANCH    = "undefined"
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
	if err != nil {
		log.Fatalf("error when reading config: %v\n", err)
	}

	// Get server hostname
	hostname := branch.GetHostname()

	// Main variables
	dbName := fmt.Sprintf("%s", branch.ParseBranchName(*refSlug))

	// Use Format for extracted file, so they don't conflicted
	current := time.Now()
	dumpFileFormat := fmt.Sprintf(current.Format("20060102.150405"))
	dumpFileDst := fmt.Sprintf("%s/dump_%s.sql", conf.GetString("dbdir"), dumpFileFormat)

	if branch.DirectoryExists(conf.GetString("storagedir")) {
		os.Chdir(conf.GetString("storagedir"))

		// Pipeline commands
		ls := exec.Command("find", ".", "-name", "*.sql.gz")
		tail := exec.Command("tail", "-1")

		// Run the pipeline
		output, stderr, err := branch.PipeLine(ls, tail)

		if err != nil {
			log.Fatalf("Error: %s", err)
		}

		// Print the stdout, if any
		//if len(output) > 0 {
		//	fmt.Printf("Output: %s\n", output)
		//}

		// Print the stderr, if any
		if len(stderr) > 0 {
			log.Fatalf("%q: (stderr)\n", stderr)
		}

		// Convert byte output to string
		dumpFileStr := string(output[:])
		dumpFileSrc := strings.TrimSpace(dumpFileStr)

		// Copy last database dbdump to dbDir
		branch.RunCommand("rsync", "-P", "-t", dumpFileSrc, conf.GetString("dbdir"))

		os.Chdir(conf.GetString("dbdir"))

		// Extract database dbdump, use time() for each extracted file *.sql
		branch.UnpackGzipFile(dumpFileSrc, dumpFileDst)

		// Prepare database
		// [user[:pass]@][protocol[(addr)]]/dbname[?p1=v1&...]
		mysqlInfo := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s",
			*mysqlUser, *mysqlPassword, *mysqlHostname, *mysqlPort, *mysqlDatabase)

		db, err := sql.Open("mysql", mysqlInfo)
		if err != nil {
			fmt.Println(err.Error())
		}
		defer db.Close()

		// make sure connection is available
		err = db.Ping()
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			log.Println("Successfully connected to MySQL!")
		}

		drop, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := drop.RowsAffected()
			log.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
			log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		create, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;", dbName))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := create.RowsAffected()
			log.Printf("MySQL: Running: CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;\n", dbName)
			log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		grant, err := db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';", dbName, dbName, dbName))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := grant.RowsAffected()
			log.Printf("MySQL: Running: GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';\n", dbName, dbName, dbName)
			log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		flush, err := db.Exec(fmt.Sprintf("FLUSH PRIVILEGES;"))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := flush.RowsAffected()
			log.Printf("MySQL: Running: FLUSH PRIVILEGES;")
			log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		// Import database dump
		branch.RunCommand("bash", "-c", fmt.Sprintf("time mysql -u%s %s < %s", *mysqlUser, dbName, dumpFileDst))

		if strings.Contains(hostname, "ees") {
			update, err := db.Exec(fmt.Sprintf("UPDATE %s.user_data SET salary = 10000, salary_proposed = 11000;", dbName))
			if err != nil {
				log.Fatalf(err.Error())
			} else {
				count, _ := update.RowsAffected()
				log.Printf("MySQL: Running: UPDATE user_data SET salary = 10000, salary_proposed = 11000;\n")
				log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
			}
		}

		// Delete database dbdump's
		branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/*.sql*", conf.GetString("dbdir")))

	} else {
		log.Fatalf("Error: No such file or directory %v\n", conf.GetString("storagedir"))
	}

}
