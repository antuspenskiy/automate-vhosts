package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"path"

	"github.com/antuspenskiy/automate-vhosts/pkg/config"
	"github.com/antuspenskiy/automate-vhosts/pkg/cmd"
	"github.com/antuspenskiy/automate-vhosts/pkg/db"
)

func main() {
	// Set the command line arguments
	var (
		refSlug   = flag.String("refslug", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		commitSha = flag.String("commitsha", "", "The commit revision for which project is built.")
	)

	// Get command line arguments
	flag.Parse()

	// Load json configuration
	conf, err := config.ReadConfig("env")
	cmd.Check(err)

	// Get server hostname
	hostname := cmd.GetHostname()

	// Variables
	hostDir := path.Join(conf.GetString("rootdir"), *refSlug)
	bxConf := path.Join(hostDir, conf.GetString("server.settings"))
	bxConn := path.Join(hostDir, conf.GetString("server.dbconn"))
	dbName := db.ParseBranchName(*refSlug)

	// Checkout to commit, run deploy commands from env.json
	if cmd.DirectoryExists(hostDir) {
		log.Printf("Directory %s exists.\n\n", hostDir)

		err = os.Chdir(hostDir)
		cmd.Check(err)

		cmd.RunCommand("bash", "-c", "git fetch --prune origin")
		cmd.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))

		cmd.Deploy(conf.GetString("server.cmd-dir-exist"))

	} else {
		log.Printf("Create directory %s.\n\n", hostDir)
		err = os.Mkdir(hostDir, 0750)
		cmd.Check(err)
		err = os.Chdir(hostDir)
		cmd.Check(err)

		booksConf := path.Join(hostDir, "env.json")

		// Create node library configuration file, need before func Deploy()
		if strings.Contains(hostname, "intranet") {
			booksTemplate := &config.BooksConfig{
				Production: config.BooksConfigNested{
					BooksEnv: config.BooksEnv{
						BaseName: dbName,
						UserName: dbName,
						Password: dbName,
						Host:     "localhost",
					},
					ExternalServerAPI: "https://127.0.0.1",
				},
				Development: config.BooksConfigNested{
					BooksEnv: config.BooksEnv{
						BaseName: dbName,
						UserName: dbName,
						Password: dbName,
						Host:     "localhost",
					},
					ExternalServerAPI: "https://127.0.0.1",
				},
			}
			booksTemplate.Write(booksConf)
			log.Printf("Library configuration %s created\n", booksConf)
			config.PrettyJson(booksTemplate)
		}
	}

	// Create environment .env for Laravel applications
	laravelConf := path.Join(hostDir, ".env")

	if strings.Contains(hostname, "ees") {
		laravelData := config.LaravelTemplate{
			AppURL:       *refSlug,
			DBDatabase:   dbName,
			DBUserName:   dbName,
			DBPassword:   dbName,
			TemplatePath: conf.GetString("server.envtmpl"),
		}
		laravelData.Write(laravelConf)
		log.Printf("Laravel environment configuration %s created\n", laravelConf)
	}

	cmd.RunCommand("bash", "-c", "git init")
	cmd.RunCommand("bash", "-c", fmt.Sprintf("git remote add -t %s -f origin %s", *refSlug, conf.GetString("server.giturl")))
	cmd.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))

	cmd.Deploy(conf.GetString("server.cmd-dir-not-exist"))

	if strings.Contains(hostname, "intranet") {
		if cmd.DirectoryExists(bxConf) || cmd.DirectoryExists(bxConn) {
		} else {
			log.Println("Run parse settings...")
			cmd.RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", bxConf, bxConf))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", bxConn, bxConn))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("php -f %s %s %s %s", conf.GetString("server.parse"), hostDir, dbName, dbName))
			log.Println("Parse complete.")
		}
	}
}
