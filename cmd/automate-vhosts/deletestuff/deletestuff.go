package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"

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

	// Variables
	hostDir := conf.GetString("rootdir") + *refSlug
	pm2Dir := conf.GetString("rootdir") + conf.GetString("server.pm2")

	// List remote branches, only 2nd row without refs/heads/
	os.Chdir(hostDir)
	gitlsRemote, err := exec.Command("bash", "-c", "sudo -u user git ls-remote --heads origin | awk '{print $2}' | sed 's/.*\\/\\(.*\\).*/\\1/'").CombinedOutput()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Printf("\nRemote Branches:\n\n%s\n", gitlsRemote)

	// List folders
	os.Chdir(conf.GetString("rootdir"))
	lsFolder, err := exec.Command("bash", "-c", "ls -d */ | grep -v 'pm2json\\|log\\|intranet' | cut -f1 -d'/'").CombinedOutput()
	if err != nil {
		os.Stderr.WriteString(err.Error())
	}
	fmt.Printf("Branches Folders:\n\n%s\n", lsFolder)

	// Convert []byte to string and split string to []string
	gitlsRemoteStr := string(gitlsRemote[:])
	gitlsRemoteStrSplit := strings.Split(gitlsRemoteStr, "\n")

	lsFolderStr := string(lsFolder[:])
	lsFolderStrSplit := strings.Split(lsFolderStr, "\n")

	// Use difference function
	diffStr := branch.Difference(lsFolderStrSplit, gitlsRemoteStrSplit)

	// Connect to MySQL
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

	// Show difference values use this values for delete db, settings and etc.
	for _, diffVal := range diffStr {
		fmt.Printf("This folder and settings will be deleted:\n%s\n\n", diffVal)

		// dbName not equal diffVal we need to parse this values in ParseBranchName
		dbName := fmt.Sprintf("%s", branch.ParseBranchName(diffVal))

		// Delete MySQL database

		drop, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := drop.RowsAffected()
			log.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
			log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		user, err := db.Exec(fmt.Sprintf("DROP USER '%s'@'localhost';", dbName))
		if err != nil {
			log.Fatalf(err.Error())
		} else {
			count, _ := user.RowsAffected()
			log.Printf("MySQL: Running: DROP USER '%s'@'localhost';\n", dbName)
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

		// Remove virtual host directory
		branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s", diffVal))

		// Remove nginx configuration file for virtual host
		branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.conf", conf.GetString("nginxdir"), diffVal))

		// Remove php-fpm configuration file for virtual host
		branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.conf", conf.GetString("fpmdir"), diffVal))

		if strings.Contains(hostname, "intranet") {
			// Remove pm2 process and configuration file for virtual host
			branch.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 delete --silent %s", diffVal))
			branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.json", pm2Dir, diffVal))
		}
	}
	// Restart nginx and php-fpm
	branch.RunCommand("bash", "-c", "systemctl restart nginx php-fpm")
}
