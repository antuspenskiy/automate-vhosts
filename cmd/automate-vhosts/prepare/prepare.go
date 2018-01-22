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

var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	flag.PrintDefaults()
}

var (
	VERSION    = "undefined"
	BUILDTIME  = "undefined"
	COMMIT    = "undefined"
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
	if err != nil {
		log.Fatalf("error when reading config: %v\n", err)
	}

	// Get server hostname
	hostname := branch.GetHostname()

	// Variables
	hostDir := conf.GetString("rootdir") + *refSlug
	bxConf := hostDir + conf.GetString("server.settings")
	bxConn := hostDir + conf.GetString("server.dbconn")
	dbName := branch.ParseBranchName(*refSlug)

	if branch.DirectoryExists(hostDir) {

		log.Printf("Directory %s exists.\n\n", hostDir)
		os.Chdir(hostDir)

		branch.RunCommand("bash", "-c", "git fetch --prune origin")
		branch.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))

		branch.Deploy(conf.GetString("server.cmd-dir-exist"))

	} else {
		log.Printf("Create directory %s.\n\n", hostDir)
		os.Mkdir(hostDir, 0750)
		os.Chdir(hostDir)

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
			data, err := json.MarshalIndent(post, "", " ")
			if err != nil {
				log.Fatalln("MarshalIndent:", err)
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
			branch.WriteStringToFile(fmt.Sprintf("%s/.env", hostDir), txt)
			log.Printf("Environemnt configuration for %s/.env created\n", hostDir)
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
