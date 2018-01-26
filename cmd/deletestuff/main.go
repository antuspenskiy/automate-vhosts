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

	// Variables
	hostDir := conf.GetString("rootdir") + *refSlug
	pm2Dir := conf.GetString("rootdir") + conf.GetString("server.pm2")

	// List remote branches, only 2nd row without refs/heads/
	err = os.Chdir(hostDir)
	branch.Check(err)
	gitlsRemote, err := exec.Command("bash", "-c", "sudo -u user git ls-remote --heads origin | awk '{print $2}' | sed 's/.*\\/\\(.*\\).*/\\1/'").CombinedOutput()
	branch.Check(err)
	fmt.Printf("\nRemote Branches:\n\n%s\n", gitlsRemote)

	// List folders
	err = os.Chdir(conf.GetString("rootdir"))
	branch.Check(err)
	lsFolder, err := exec.Command("bash", "-c", "ls -d */ | grep -v 'pm2json\\|log\\|intranet\\|default' | cut -f1 -d'/'").CombinedOutput()
	branch.Check(err)
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

	// Show difference values use this values for delete db, settings and etc.
	for _, diffVal := range diffStr {
		fmt.Printf("This folder and settings will be deleted:\n%s\n\n", diffVal)

		// dbName not equal diffVal we need to parse this values in ParseBranchName
		dbName := branch.ParseBranchName(diffVal)

		// Delete MySQL database

		numdrop, err := branch.DropDB(db, dbName)
		branch.Check(err)
		log.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numdrop)

		numdropuser, err := branch.DropUser(db, dbName)
		branch.Check(err)
		log.Printf("MySQL: Running: DROP USER '%s'@'localhost';\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numdropuser)

		numflush, err := branch.FlushPriv(db)
		branch.Check(err)
		log.Printf("MySQL: Running: FLUSH PRIVILEGES;")
		log.Printf("MySQL: Query OK, %d rows affected\n\n", numflush)

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
