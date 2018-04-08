package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/antuspenskiy/automate-vhosts/pkg/config"
	"github.com/antuspenskiy/automate-vhosts/pkg/cmd"
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
		refSlug = flag.String("refslug", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
	)

	// Get command line arguments
	flag.Parse()

	// Load json configuration
	conf, err := config.ReadConfig("env")
	cmd.Check(err)

	// Get server hostname
	hostname := cmd.GetHostname()

	// Variables
	hostDir := conf.GetString("rootdir") + *refSlug
	pm2Dir := conf.GetString("server.pm2")

	// Generate random port for nginx, php-fpm and pm2 configuration
	portphp := config.RandomTCPPort()
	portnode := config.RandomTCPPort()

	// Create nginx configuration for virtual host
	if cmd.DirectoryExists(fmt.Sprintf("%s/%s.conf", conf.GetString("nginxdir"), *refSlug)) {
		log.Printf("Nginx configuration for %s/%s.conf exist\n", conf.GetString("nginxdir"), *refSlug)
	} else {

		// TODO: Refactor to func
		nginxData := config.NginxTemplate{
			ServerName: fmt.Sprintf("%s.%s", *refSlug, conf.GetString("subdomain")),
			PortPhp:    portphp,
			PortNode:   portnode,
			RefSlug:    *refSlug,
		}

		txt := config.ParseTemplate(conf.GetString("server.nginxtmpl"), nginxData)
		err = config.WriteStringToFile(fmt.Sprintf("%s/%s.conf", conf.GetString("nginxdir"), *refSlug), txt)
		cmd.Check(err)
		log.Printf("Nginx configuration for %s/%s.conf created\n", conf.GetString("nginxdir"), *refSlug)
	}

	// Create php-fpm configuration
	if cmd.DirectoryExists(fmt.Sprintf("%s/%s.conf", conf.GetString("fpmdir"), *refSlug)) {
		log.Printf("Php-fpm configuration for %s/%s.conf exist\n", conf.GetString("fpmdir"), *refSlug)
	} else {
		fileHandle, err := os.Create(fmt.Sprintf("%s/%s.conf", conf.GetString("fpmdir"), *refSlug))
		if err != nil {
			log.Println("Error creating configuration file:", err)
			return
		}
		// TODO: Refactor to func
		writer := bufio.NewWriter(fileHandle)
		defer func() {
			err = fileHandle.Close()
			cmd.Check(err)
		}()

		fmt.Fprintln(writer, fmt.Sprintf("[%s]", *refSlug))
		fmt.Fprintln(writer, fmt.Sprintf("listen = 127.0.0.1:%d", portphp))
		fmt.Fprintln(writer, "user = user")
		fmt.Fprintln(writer, "pm = static")
		fmt.Fprintln(writer, "pm.max_children = 2")
		fmt.Fprintln(writer, "pm.max_requests = 500")
		fmt.Fprintln(writer, "request_terminate_timeout = 65m")
		fmt.Fprintln(writer, "php_admin_value[max_execution_time] = 300")
		fmt.Fprintln(writer, "php_admin_value[sendmail_path] = false")

		if strings.Contains(hostname, "intranet") {
			fmt.Fprintln(writer, "php_admin_value[mbstring.func_overload] = 4")
		}

		err = writer.Flush()
		cmd.Check(err)

		log.Printf("Php-fpm configuration for %s/%s.conf created\n", conf.GetString("fpmdir"), *refSlug)

		cmd.RunCommand("bash", "-c", "systemctl restart nginx php-fpm")
	}

	// Create pm2 configuration for test-intranet
	if strings.Contains(hostname, "intranet") {
		if cmd.DirectoryExists(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug)) {
			// Don't reload process, delete it and start again
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 describe %s", *refSlug))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 delete -s %s || :", *refSlug))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 start %s/%s.json", pm2Dir, *refSlug))
		} else {
			var buf bytes.Buffer
			pm2Conf := &config.Post{
				Apps: []config.App{
					{
						ExecMode: "fork_mode",
						Script:   "tools/run.js",
						Args:     []string{"start"},
						Name:     *refSlug,
						Cwd:      hostDir,
						Env: config.Env{
							Port:    portnode,
							NodeEnv: "development",
						},
						ErrorFile: fmt.Sprintf("log/%s.err.log", *refSlug),
						OutFile:   fmt.Sprintf("log/%s.out.log", *refSlug),
					},
				},
			}
			config.EncodeTo(&buf, pm2Conf)

			// Pretty print json file
			data, err := json.MarshalIndent(pm2Conf, "", " ")
			if err != nil {
				log.Fatalln("MarshalIndent:", err)
			}
			log.Printf("PM2 JSON configuration created:\n%s", data)

			if fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug) != "" {
				if err = ioutil.WriteFile(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug), data, 0644); err != nil {
					log.Fatalln("WriteFile:", err)
				}

				// Chown pm2 file with user.user permissions
				err = os.Chown(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug), 1000, 1000)
				cmd.Check(err)

				// Start pm2 process
				cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 start %s/%s.json", pm2Dir, *refSlug))

			} else {
				log.Printf("%s\n", string(data))
			}
		}
	}
}
