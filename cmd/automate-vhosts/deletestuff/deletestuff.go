package main

import (
	"flag"
	"fmt"
	"database/sql"
	"log"

	"github.com/antuspenskiy/automate-vhosts/pkg/branch"
)

var (
	Version     = "undefined"
	BuildTime   = "undefined"
	GitHash     = "undefined"
)

func main() {
	fmt.Printf("Version    : %s\n", Version)
	fmt.Printf("Git Hash   : %s\n", GitHash)
	fmt.Printf("Build Time : %s\n\n", BuildTime)

	// Set the command line arguments
	var (
		configDir     = "/opt/scripts/configs/config.json"
		refSlug       = flag.String("CI_COMMIT_REF_SLUG", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		mysqlUser     = flag.String("user", "", "Name of your database user.")
		mysqlPassword = flag.String("password", "", "Name of your database user password.")
		mysqlHostname = flag.String("hostname", "localhost", "Name of your database hostname.")
		mysqlPort     = flag.String("port", "3306", "Name of your database port.")
		mysqlDatabase = flag.String("database", "", "Name of your database.")
	)

	c, _ := branch.LoadConfiguration(configDir)

	// Get command line arguments
	flag.Parse()

	hostDir := c.RootDir + *refSlug
	pm2Dir := c.RootDir + c.Testing.PmDir
	dbName := fmt.Sprintf("%s", branch.PassArguments(*refSlug))

	// Remove virtual host directory
	branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s", hostDir))

	// Remove nginx configuration file for virtual host
	branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.conf", c.Testing.NginxSettings, *refSlug))

	// Remove php-fpm configuration file for virtual host
	branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.conf", c.Testing.PoolSettings, *refSlug))

	// Remove pm2 process and configuration file for virtual host
	branch.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 delete --silent %s", *refSlug))
	branch.RunCommand("bash", "-c", fmt.Sprintf("rm -fr %s/%s.json", pm2Dir, *refSlug))

	// Delete MySQL database
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

	dropuser, err := db.Exec(fmt.Sprintf("DROP USER '%s'@'localhost';", dbName))
	if err != nil {
		log.Fatalf(err.Error())
	} else {
		count, _ := dropuser.RowsAffected()
		log.Printf("MySQL: Running: DROP USER '%s'@'localhost';\n", dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
	}

	dropprivs, err := db.Exec(fmt.Sprintf("REVOKE ALL PRIVILEGES ON %s.* FROM '%s'@'localhost';", dbName, dbName))
	if err != nil {
		log.Fatalf(err.Error())
	} else {
		count, _ := dropprivs.RowsAffected()
		log.Printf("MySQL: Running: REVOKE ALL PRIVILEGES ON %s.* FROM '%s'@'localhost';\n", dbName, dbName)
		log.Printf("MySQL: Query OK, %d rows affected\n\n", count)
	}

	// Restart nginx and php-fpm
	branch.RunCommand("bash", "-c", "systemctl restart nginx php-fpm")
}
