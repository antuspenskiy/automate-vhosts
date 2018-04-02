package branch

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"text/template"
	"time"
	"archive/tar"
	"path/filepath"

	"github.com/spf13/viper"
)

// LibPost represents a struct for library configuration
type LibPost struct {
	P LibConfiguration `json:"production"`
	D LibConfiguration `json:"development"`
}

// BaseConfig represent a struct for database settings
type BaseConfig struct {
	BaseName string `json:"BASE_NAME"`
	UserName string `json:"USER_NAME"`
	Password string `json:"PASSWORD"`
	Host     string `json:"HOST"`
}

// LibConfiguration represent a struct for library configuration
type LibConfiguration struct {
	BaseConfig        BaseConfig `json:"BASE_CONFIG"`
	ExternalServerAPI string     `json:"EXTERNAL_SERVER_API"`
}

// Post represent pm2 struct which contains an array of variables
type Post struct {
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

// NginxTemplate represent struct for nginx configuration
type NginxTemplate struct {
	ServerName string
	PortPhp    int
	PortNode   int
	RefSlug    string
}

// LaravelTemplate represent struct for laravel environment configuration
type LaravelTemplate struct {
	AppURL     string
	DBDatabase string
	DBUserName string
	DBPassword string
}

const (
	defaultFailedCode  = 1
	minTCPPort         = 0
	maxTCPPort         = 9000
	maxReservedTCPPort = 8080
	maxRandTCPPort     = maxTCPPort - (maxReservedTCPPort + 1)
)

var (
	tcpPortRand = rand.New(rand.NewSource(time.Now().UnixNano()))
)

// ParseBranchName parse branch name symbols and lenght
func ParseBranchName(name string) string {

	// Remove all Non-Alphanumeric Characters from a NameBranch
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatalf("Error: %s", err)
	}

	branchString := reg.ReplaceAllString(name, "_")

	// User name (should be no longer than 32) for Percona Server
	if len(branchString) > 32 {
		branchCut := branchString[0:32]
		fmt.Printf("A string of %s becomes %s \n", name, branchCut)
		return branchCut
	}
	fmt.Printf("A string of %s becomes %s \n", name, branchString)
	return branchString
}

// ReadConfig read json environment file from directory
func ReadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(filename)
	v.AddConfigPath("/opt/scripts/config")
	v.AutomaticEnv()
	err := v.ReadInConfig()
	return v, err
}

// RunCommand exec command and print stdout,stderr and exitCode
func RunCommand(name string, args ...string) (stdout string, stderr string, exitCode int) {
	log.Println("run command:", name, args)
	var outbuf, errbuf bytes.Buffer
	cmd := exec.Command(name, args...)
	cmd.Stdout = &outbuf
	cmd.Stderr = &errbuf

	err := cmd.Run()
	stdout = outbuf.String()
	stderr = errbuf.String()

	if err != nil {
		// try to get the exit code
		if exitError, ok := err.(*exec.ExitError); ok {
			ws := exitError.Sys().(syscall.WaitStatus)
			exitCode = ws.ExitStatus()
		} else {
			// This will happen (in OSX) if `name` is not available in $PATH,
			// in this situation, exit code could not be get, and stderr will be
			// empty string very likely, so we use the default fail code, and format err
			// to string and set to stderr
			log.Printf("Could not get exit code for failed program: %v, %v", name, args)
			exitCode = defaultFailedCode
			if stderr == "" {
				stderr = err.Error()
			}
		}
	} else {
		// success, exitCode should be 0 if go is ok
		ws := cmd.ProcessState.Sys().(syscall.WaitStatus)
		exitCode = ws.ExitStatus()
	}
	if exitCode != 0 {
		log.Fatalf("command result, stdout: %v, stderr: %v, exitCode: %v", stdout, stderr, exitCode)
	}
	log.Printf("command result, stdout: %v, stderr: %v, exitCode: %v", stdout, stderr, exitCode)
	return
}

// DirectoryExists returns true if a directory(or file) exists, otherwise false
func DirectoryExists(dir string) bool {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return false
	}
	return true
}

func FilePathWalkDir(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// Gunzip extract sql.gz archive files to destination folder
//func Gunzip(gzfile string, gzfiledest string) {
//	// Open gzip file that we want to uncompress
//	// The file is a reader, but we could use any
//	// data source. It is common for web servers
//	// to return gzipped contents to save bandwidth
//	// and in that case the data is not in a file
//	// on the file system but is in a memory buffer
//	gzipFile, err := os.Open(gzfile)
//	if err != nil {
//		log.Fatal(err)
//	}
//	// Create a gzip reader on top of the file reader
//	// Again, it could be any type reader though
//	gzipReader, err := gzip.NewReader(gzipFile)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer func() {
//		err = gzipReader.Close()
//		if err != nil {
//			log.Fatalln(err)
//		}
//	}()
//
//	// Uncompress to a writer. We'll use a file writer
//	outfileWriter, err := os.Create(gzfiledest)
//	if err != nil {
//		log.Fatal(err)
//	}
//	defer func() {
//		err = outfileWriter.Close()
//		if err != nil {
//			log.Fatalln(err)
//		}
//	}()
//
//	// Copy contents of gzipped file to output file
//	_, err = io.Copy(outfileWriter, gzipReader)
//	if err != nil {
//		log.Fatal(err)
//	}
//}

// ExtractTarGz extracting *.tar.gz archives to destination folder
func ExtractTarGz(tarFile string, tarExtractDst string) {
	f, err := os.Open(tarFile)
	if err != nil {
		log.Fatal("Open file failed")
	}
	defer f.Close()

	gzf, err := gzip.NewReader(f)
	if err != nil {
		log.Fatal("Couldn't create gzip reader")
	}
	defer f.Close()

	tarReader := tar.NewReader(gzf)

	for true {

		_, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalf("Next() failed: %s", err.Error())
		}

		outFile, err := os.Create(tarExtractDst)
		if err != nil {
			log.Fatalf("Create() failed: %s", err.Error())
		}
		log.Printf("Create file %s", tarExtractDst)
		defer outFile.Close()

		if _, err := io.Copy(outFile, tarReader); err != nil {
			log.Fatalf("Copy() failed: %s", err.Error())
		}
		log.Println("File extracted succefully")
	}
}

// DeleteFile delete file
func DeleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		log.Fatalf("Error: %v\n\n", err)
	}
	log.Printf("File %s deleted. \n", path)
}

// Deploy deploy commands for virtual hosts
func Deploy(conf string) []string {
	cmd := conf
	commands := strings.Split(cmd, ",")

	for _, command := range commands {
		RunCommand("bash", "-c", command)
	}
	return commands
}

// PipeLine run pipline commands
func PipeLine(cmds ...*exec.Cmd) (pipeLineOutput, collectedStandardError []byte, pipeLineError error) {
	// Require at least one command
	if len(cmds) < 1 {
		return nil, nil, nil
	}

	// Collect the output from the command(s)
	var output bytes.Buffer
	var stderr bytes.Buffer

	last := len(cmds) - 1
	for i, cmd := range cmds[:last] {
		var err error
		// Connect each command's stdin to the previous command's stdout
		if cmds[i+1].Stdin, err = cmd.StdoutPipe(); err != nil {
			return nil, nil, err
		}
		// Connect each command's stderr to a buffer
		cmd.Stderr = &stderr
	}

	// Connect the output and error for the last command
	cmds[last].Stdout, cmds[last].Stderr = &output, &stderr

	// Start each command
	for _, cmd := range cmds {
		if err := cmd.Start(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Wait for each command to complete
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			return output.Bytes(), stderr.Bytes(), err
		}
	}

	// Return the pipeline output and the collected standard error
	return output.Bytes(), stderr.Bytes(), nil
}

// ParseSettings parse settings for intranet-test virtual hosts
func ParseSettings(bxconf string, bxconn string, parse string, hostdir string, dbname string) {
	RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", bxconf, bxconf))
	RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", bxconn, bxconn))
	RunCommand("bash", "-c", fmt.Sprintf("php -f %s %s %s %s", parse, hostdir, dbname, dbname))
}

// IsTCPPortAvailable returns a flag indicating whether or not a TCP port is
// available.
func IsTCPPortAvailable(port int) bool {
	if port < minTCPPort || port > maxTCPPort {
		return false
	}
	conn, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	defer func() {
		err = conn.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()
	return true
}

// RandomTCPPort gets a free, random TCP port between 1025-65535. If no free
// ports are available -1 is returned.
func RandomTCPPort() int {
	for i := maxReservedTCPPort; i < maxTCPPort; i++ {
		p := tcpPortRand.Intn(maxRandTCPPort) + maxReservedTCPPort + 1
		if IsTCPPortAvailable(p) {
			return p
		}
	}
	return -1
}

// WriteStringToFile save configuration files in filesystem
func WriteStringToFile(filepath, s string) error {
	fo, err := os.Create(filepath)
	if err != nil {
		log.Fatalf("Error: %v\n\n", err)
		return err
	}
	defer func() {
		err = fo.Close()
		if err != nil {
			log.Fatalln(err)
		}
	}()

	_, err = io.Copy(fo, strings.NewReader(s))
	if err != nil {
		log.Fatalf("Error: %v\n\n", err)
		return err
	}
	return nil
}

// ParseTemplate is parse struct variables in different templates for configuration files
func ParseTemplate(templateFileName string, data interface{}) string {
	t, err := template.ParseFiles(templateFileName)
	if err != nil {
		log.Println(err)
	}
	buf := new(bytes.Buffer)
	if err = t.Execute(buf, data); err != nil {
		log.Println(err)
	}
	return buf.String()
}

// EncodeTo save configuration files in json
func EncodeTo(w io.Writer, i interface{}) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(i); err != nil {
		log.Fatalf("failed encoding to writer: %s", err)
	}
}

// Difference returns the elements in a that aren't in b
func Difference(a, b []string) []string {
	mb := map[string]bool{}
	for _, x := range b {
		mb[x] = true
	}
	ab := []string{}
	for _, x := range a {
		if _, ok := mb[x]; !ok {
			ab = append(ab, x)
		}
	}
	return ab
}

// GetHostname get os.Hostname
func GetHostname() string {
	hostname, err := os.Hostname()
	if err != nil {
		log.Fatalf("error get server hostname: %v\n", err)
	}
	return hostname
}

// Check error checking
func Check(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// DropSalary clear data before using database
func DropSalary(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("UPDATE %s.user_data SET salary = 10000, salary_proposed = 11000;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DropDB drop MySQL database
func DropDB(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// DropUser drop user
func DropUser(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("DROP USER '%s'@'localhost';", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// CreateDB create MySQL database
func CreateDB(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;", dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// GrantUserPriv grant user privileges to MySQL DB
func GrantUserPriv(db *sql.DB, dbname string) (int64, error) {
	res, err := db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';", dbname, dbname, dbname))
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// FlushPriv flush the privileges
func FlushPriv(db *sql.DB) (int64, error) {
	res, err := db.Exec("FLUSH PRIVILEGES;")
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
