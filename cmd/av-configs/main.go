package main

import (
	"flag"
	"fmt"
	"log"
	"strings"
	"path"

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
	hostName := cmd.GetHostname()

	// Variables
	hostDir := path.Join(conf.GetString("rootdir"), *refSlug)

	// Generate random port for nginx, php-fpm and pm2 configuration
	portPhp := config.RandomTCPPort()
	portNode := config.RandomTCPPort()

	// Create nginx configuration for virtual host
	nginxConf := path.Join(conf.GetString("nginxdir"), *refSlug+".conf")

	if cmd.DirectoryExists(nginxConf) {
		log.Printf("Nginx configuration %s exist!\n", nginxConf)
	} else {

		// TODO: Refactor to func
		nginxData := config.NginxTemplate{
			ServerName:   fmt.Sprintf("%s.%s", *refSlug, conf.GetString("subdomain")),
			PortPhp:      portNode,
			PortNode:     portPhp,
			RefSlug:      *refSlug,
			TemplatePath: conf.GetString("server.nginxtmpl"),
		}

		err = nginxData.Write(nginxConf)
		cmd.Check(err)
		log.Printf("Nginx configuration %s created\n", nginxConf)
	}

	// Create php-fpm configuration
	fpmConf := path.Join(conf.GetString("fpmdir"), *refSlug+".conf")

	if cmd.DirectoryExists(fpmConf) {
		log.Printf("Php-fpm configuration %s exist!\n", fpmConf)
	} else {
		m := make(map[string]string)
		m["listen"] = fmt.Sprintf("127.0.0.1:%d\n", portPhp)
		m["user"] = "user"
		m["pm"] = "static"
		m["pm.max_children"] = "2"
		m["pm.max_requests"] = "500"
		m["request_terminate_timeout"] = "65m"
		m["php_admin_value[max_execution_time]"] = "300"
		m["php_admin_value[sendmail_path]"] = "false"

		if strings.Contains(hostName, "intranet") {
			m["php_admin_value[mbstring.func_overload]"] = "4"
		}

		fpm := config.FpmConfig{
			Section: *refSlug,
			Params:  m,
		}

		fpm.Write(fpmConf)
		log.Printf("Php-fpm configuration %s created\n", fpmConf)
		cmd.RunCommand("bash", "-c", "systemctl restart nginx php-fpm")
	}

	// Create pm2 configuration for test-intranet
	pm2Conf := path.Join(conf.GetString("server.pm2"), *refSlug+".json")

	if strings.Contains(hostName, "intranet") {
		if cmd.DirectoryExists(pm2Conf) {
			log.Printf("Pm2 configuration %s exist!\n", pm2Conf)

			// Don't reload process, delete it and start again
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 describe %s", *refSlug))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 delete -s %s || :", *refSlug))
			cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 start %s", pm2Conf))
		} else {
			pm2Data := config.PM2Config{
				Apps: []config.App{
					{
						ExecMode: "fork_mode",
						Script:   "tools/run.js",
						Args:     []string{"start"},
						Name:     *refSlug,
						Cwd:      hostDir,
						Env: config.Env{
							Port:    portNode,
							NodeEnv: "development",
						},
						ErrorFile: fmt.Sprintf("log/%s.err.log", *refSlug),
						OutFile:   fmt.Sprintf("log/%s.out.log", *refSlug),
					},
				},
			}

			if pm2Data.Write(pm2Conf) {
				log.Printf("Pm2 configuration created:\n%s", pm2Data.PrettyJson())
				// Start pm2 process
				cmd.RunCommand("bash", "-c", fmt.Sprintf("sudo -u user pm2 start %s", pm2Conf))
			} else {
				log.Println("Pm2 configuration not created!")
			}

		}
	}
}
