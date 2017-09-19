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
	)

	c, _ := branch.LoadConfiguration(configDir)

	// Get command line arguments
	flag.Parse()

	hostDir := c.RootDir + *refSlug
	pm2Dir := c.RootDir + c.Testing.PmDir

	// Generate random port for nginx, php-fpm and pm2 configuration
	portphp := branch.RandomTCPPort()
	portnode := branch.RandomTCPPort()

	// Create nginx configuration for virtual host
	if branch.DirectoryExists(fmt.Sprintf("%s/%s.conf", c.Testing.NginxSettings, *refSlug)) {
	} else {
		templateData := branch.NginxTemplate{
			fmt.Sprintf("%s.%s", *refSlug, c.Testing.Hostname),
			portphp,
			portnode,
			fmt.Sprintf("%s", *refSlug),
		}

		txt := branch.ParseTemplate(fmt.Sprintf("%s", c.Testing.NginxTemplate), templateData)
		branch.WriteStringToFile(fmt.Sprintf("%s/%s.conf", c.Testing.NginxSettings, *refSlug), txt)

	}

	// Create php-fpm configuration for virtual host
	if branch.DirectoryExists(fmt.Sprintf("%s/%s.conf", c.Testing.PoolSettings, *refSlug)) {
	} else {
		fileHandle, err := os.Create(fmt.Sprintf("%s/%s.conf", c.Testing.PoolSettings, *refSlug))
		if err != nil {
			log.Println("Error creating configuration file:", err)
			return
		}
		writer := bufio.NewWriter(fileHandle)
		defer fileHandle.Close()

		fmt.Fprintln(writer, fmt.Sprintf("[%s]", *refSlug))
		fmt.Fprintln(writer, fmt.Sprintf("listen = 127.0.0.1:%d", portphp))
		fmt.Fprintln(writer, "user = user")
		fmt.Fprintln(writer, "pm = static")
		fmt.Fprintln(writer, "pm.max_children = 2")
		fmt.Fprintln(writer, "pm.max_requests = 500")
		fmt.Fprintln(writer, "request_terminate_timeout = 65m")
		fmt.Fprintln(writer, "php_admin_value[max_execution_time] = 300")
		fmt.Fprintln(writer, "php_admin_value[mbstring.func_overload] = 4")
		fmt.Fprintln(writer, "php_admin_value[sendmail_path] = false")
		writer.Flush()

		log.Printf("php-fpm configuration %s/%s.conf created\n", c.Testing.PoolSettings, *refSlug)

		branch.RunCommand("bash", "-c", "systemctl restart nginx php-fpm")
	}

	// Create pm2 configuration for virtual host
	if branch.DirectoryExists(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug)) {
		// Reload pm2 process
		branch.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 gracefulReload %s --update-env development", *refSlug))
	} else {
		var buf bytes.Buffer
		post := &branch.Post{
			Apps: []branch.App{
				{
					"fork_mode",
					"tools/run.js",
					[]string{"start"},
					fmt.Sprintf("%s", *refSlug),
					hostDir,
					branch.Env{
						portnode,
						"development",
					},
					fmt.Sprintf("log/%s.err.log", *refSlug),
					fmt.Sprintf("log/%s.out.log", *refSlug),
				},
			},
		}
		branch.EncodeTo(&buf, post)

		// Pretty print json file
		data, err := json.MarshalIndent(post, "", " ")
		if err != nil {
			log.Fatalln("MarshalIndent:", err)
		}
		log.Printf("pm2 json configuration created:\n%s", data)

		if fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug) != "" {
			if err = ioutil.WriteFile(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug), data, 0644); err != nil {
				log.Fatalln("WriteFile:", err)
			}

			// Chown pm2 file with user.user permissions
			os.Chown(fmt.Sprintf("%s/%s.json", pm2Dir, *refSlug), 1000, 1000)

			// Start pm2 process
			branch.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 start %s/%s.json", pm2Dir, *refSlug))

		} else {
			log.Printf("%s\n", string(data))
		}
	}
}
