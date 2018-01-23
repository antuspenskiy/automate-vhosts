package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

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
	fmt.Printf("Branch 	   : %s\n\n", BRANCH)

	// Set the command line arguments
	var (
		refSlug   = flag.String("refslug", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		commitSha = flag.String("commitsha", "", "The commit revision for which project is built.")
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
	bxConf := hostDir + conf.GetString("server.settings")
	bxConn := hostDir + conf.GetString("server.dbconn")
	dbName := branch.ParseBranchName(*refSlug)

	if branch.DirectoryExists(hostDir) {

		log.Printf("Directory %s exists.\n\n", hostDir)
		err = os.Chdir(hostDir)
		branch.Check(err)

		branch.RunCommand("bash", "-c", "git fetch --prune origin")
		branch.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))

		branch.Deploy(conf.GetString("server.cmd-dir-exist"))

	} else {
		log.Printf("Create directory %s.\n\n", hostDir)
		err = os.Mkdir(hostDir, 0700)
		branch.Check(err)
		err = os.Chdir(hostDir)
		branch.Check(err)

		// Create library configuration file for intranet-test, need before func Deploy()
		if strings.Contains(hostname, "intranet") {
			var buf bytes.Buffer
			post := &branch.LibPost{
				P: branch.LibConfiguration{
					BaseConfig: branch.BaseConfig{
						BaseName: dbName,
						UserName: dbName,
						Password: dbName,
						Host:     "localhost",
					},
					ExternalServerAPI: "https://127.0.0.1",
				},
				D: branch.LibConfiguration{
					BaseConfig: branch.BaseConfig{
						BaseName: dbName,
						UserName: dbName,
						Password: dbName,
						Host:     "localhost",
					},
					ExternalServerAPI: "https://127.0.0.1",
				},
			}
			branch.EncodeTo(&buf, post)

			// Pretty print json file
			data, cerr := json.MarshalIndent(post, "", " ")
			if cerr != nil {
				log.Fatalln("MarshalIndent:", cerr)
			}
			log.Printf("Library JSON configuration created:\n%s", data)

			if fmt.Sprintf("%s/env.json", hostDir) != "" {
				if err = ioutil.WriteFile(fmt.Sprintf("%s/env.json", hostDir), data, 0644); err != nil {
					log.Fatalln("WriteFile:", err)
				}
			}
		}

		// Create environment .env for Laravel applications
		if strings.Contains(hostname, "ees") {
			laravelData := branch.LaravelTemplate{
				AppURL:     *refSlug,
				DBDatabase: dbName,
				DBUserName: dbName,
				DBPassword: dbName,
			}

			log.Printf("Create environment file %s/.env\n", hostDir)

			txt := branch.ParseTemplate(conf.GetString("server.envtmpl"), laravelData)
			err = branch.WriteStringToFile(fmt.Sprintf("%s/.env", hostDir), txt)
			branch.Check(err)
			log.Printf("Environment configuration for %s/.env created\n", hostDir)
		}

		branch.RunCommand("bash", "-c", "git init")
		branch.RunCommand("bash", "-c", fmt.Sprintf("git remote add -t %s -f origin %s", *refSlug, conf.GetString("server.giturl")))
		branch.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))

		branch.Deploy(conf.GetString("server.cmd-dir-not-exist"))
	}

	if strings.Contains(hostname, "intranet") {
		if branch.DirectoryExists(bxConf) || branch.DirectoryExists(bxConn) {
		} else {
			log.Println("Run parse settings...")
			branch.ParseSettings(bxConf, bxConn, conf.GetString("server.parse"), hostDir, dbName)
			log.Println("Parse complete.")
		}
	}
}
