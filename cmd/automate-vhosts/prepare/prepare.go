package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"

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
		configDir = "/opt/scripts/configs/config.json"
		refSlug   = flag.String("CI_COMMIT_REF_SLUG", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		refName   = flag.String("CI_COMMIT_REF_NAME", "", "The branch or tag name for which project is built.")
		commitSha = flag.String("CI_COMMIT_SHA", "", "The commit revision for which project is built.")
	)

	c, _ := branch.LoadConfiguration(configDir)

	// Get command line arguments
	flag.Parse()

	hostDir := c.RootDir + *refSlug
	settings := hostDir + c.Testing.SettingsFile
	dbconn := hostDir + c.Testing.DBConnFile
	dbName := fmt.Sprintf("%s", branch.PassArguments(*refSlug))

	if branch.DirectoryExists(hostDir) {

		log.Printf("Directory %s exists.\n\n", hostDir)
		os.Chdir(hostDir)

		branch.RunCommand("bash", "-c", fmt.Sprintf("git fetch --prune origin"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))
		branch.RunCommand("bash", "-c", fmt.Sprintf("composer install --no-dev --no-progress"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn clean"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn install --no-progress"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower install"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower prune"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn build"))

	} else {

		log.Printf("Create directory %s.\n\n", hostDir)
		os.Mkdir(fmt.Sprintf("%s", hostDir), 0750)
		os.Chdir(hostDir)

		// Create library configuration file

		var buf bytes.Buffer
		post := &branch.LibPost{
			branch.LibConfiguration{
				branch.BaseConfig{
					dbName,
					dbName,
					dbName,
					"localhost",
				},
				"https://127.0.0.1",
			},
			branch.LibConfiguration{
				branch.BaseConfig{
					dbName,
					dbName,
					dbName,
					"localhost",
				},
				"https://127.0.0.1",
			},
		}
		branch.EncodeTo(&buf, post)

		// Pretty print json file
		data, err := json.MarshalIndent(post, "", " ")
		if err != nil {
			log.Fatalln("MarshalIndent:", err)
		}
		log.Printf("library json configuration created:\n%s", data)

		if fmt.Sprintf("%s/env.json", hostDir) != "" {
			if err = ioutil.WriteFile(fmt.Sprintf("%s/env.json", hostDir), data, 0644); err != nil {
				log.Fatalln("WriteFile:", err)
			}
		}

		branch.RunCommand("bash", "-c", fmt.Sprintf("git init"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("git remote add -t %s -f origin %s", *refSlug, c.GitUrl))
		branch.RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))
		branch.RunCommand("bash", "-c", fmt.Sprintf("composer install --no-dev --no-progress"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn clean"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn install --no-progress"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower install"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower prune"))
		branch.RunCommand("bash", "-c", fmt.Sprintf("yarn build"))
	}

	if branch.DirectoryExists(settings) || branch.DirectoryExists(dbconn) {
	} else {

		log.Println("Run parse settings...")

		branch.RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", settings, settings))
		branch.RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", dbconn, dbconn))
		branch.RunCommand("bash", "-c", fmt.Sprintf("php -f %s %s %s %s", c.Testing.ParseSettings, hostDir, *refName, *refName))

		log.Println("Parse complete.")
	}
}
