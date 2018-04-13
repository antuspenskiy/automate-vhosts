package config

import (
	"log"
	"os"
)

// PM2Config represent pm2 struct which contains an array of variables
type PM2Config struct {
	Apps []App `json:"apps"`
}

// App represent struct for pm2 json configuration
type App struct {
	ExecMode  string   `json:"exec_mode"`
	Script    string   `json:"script"`
	Args      []string `json:"args"`
	Name      string   `json:"name"`
	Cwd       string   `json:"cwd"`
	Env       Env      `json:"env"`
	ErrorFile string   `json:"error_file"`
	OutFile   string   `json:"out_file"`
}

// Env represent struct for port generator
type Env struct {
	Port    int    `json:"PORT"`
	NodeEnv string `json:"NODE_ENV"`
}

// Write create pm2 json configuration and file permissions
func (p *PM2Config) Write(path string) bool {
	WriteJSONToFile(path, p)
	err := os.Chown(path, 1000, 1000)
	if err != nil {
		log.Fatal(err)
	}
	return err != nil
}
