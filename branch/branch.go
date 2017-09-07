package branch

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"
	"log"
	"syscall"

	_ "github.com/go-sql-driver/mysql"
)

// Configuration represents a struct for global variables and environments
type Configuration struct {
	Testing struct {
		Env           string `json:"env"`
		SettingsFile  string `json:"settings"`
		DBConnFile    string `json:"dbconn"`
		ParseSettings string `json:"parse"`
	} `json:"testing"`
	Production struct {
		Env      string `json:"env"`
		Hostname string `json:"hostname"`
	} `json:"production"`
	RootDir     string `json:"rootDir"`
	DatabaseDir string `json:"dbDir"`
	StorageDir  string `json:"storageDir"`
}

const (
	defaultFailedCode = 1
)

var (
	gitUrl    = "git@your_git_url"
	configDir = "/opt/scripts/config/config.json"
)

// PassArguments pass branch name
func PassArguments() string {

	NameBranch := flag.String("branch", "1-test-branch", "Branch name")
	flag.Parse()
	flag.Args()
	fmt.Printf("Output: Branch name is %q.", *NameBranch)

	NameBranchToString := *NameBranch

	// Remove all Non-Alphanumeric Characters from a NameBranch
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		fmt.Printf("Error: %s", err)
	}
	processedBranchString := reg.ReplaceAllString(NameBranchToString, "_")

	fmt.Printf("A Branch name of %s becomes %s. \n\n", NameBranchToString, processedBranchString)

	return processedBranchString
}

// LoadConfiguration load JSON configuration file
func LoadConfiguration(file string) (Configuration, error) {
	var config Configuration
	configFile, err := os.Open(file)
	defer configFile.Close()
	if err != nil {
		fmt.Println(err.Error())
	}
	jsonParser := json.NewDecoder(configFile)
	jsonParser.Decode(&config)
	return config, err
}

// ExecCmd run commands
//func ExecCmd(path string, args ...string) {
//	fmt.Printf("Running: %q %q\n", path, strings.Join(args, " "))
//	cmd := exec.Command(path, args...)
//	output, err := cmd.CombinedOutput()
//	if err != nil {
//		fmt.Printf(fmt.Sprint(err) + ": " + string(output))
//		return
//	} else {
//		fmt.Println(string(output))
//
//		//bs, err := cmd.CombinedOutput()
//		//
//		//fmt.Printf("Output: %s\n", bs)
//		//fmt.Printf("Error: %v\n\n", err)
//	}
//}

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

// UnpackGzipFile extract sql.gz file
func UnpackGzipFile(gzFilePath, dstFilePath string) (int64, error) {
	gzFile, err := os.Open(gzFilePath)
	if err != nil {
		return 0, fmt.Errorf("Failed to open file %s for unpack: %s", gzFilePath, err)
	}
	dstFile, err := os.OpenFile(dstFilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0660)
	if err != nil {
		return 0, fmt.Errorf("Failed to create destination file %s for unpack: %s", dstFilePath, err)
	}

	ioReader, ioWriter := io.Pipe()

	go func() { // goroutine leak is possible here
		gzReader, _ := gzip.NewReader(gzFile)
		// it is important to close the writer or reading from the other end of the
		// pipe or io.copy() will never finish
		defer func() {
			gzFile.Close()
			gzReader.Close()
			ioWriter.Close()
		}()

		io.Copy(ioWriter, gzReader)
	}()

	written, err := io.Copy(dstFile, ioReader)
	if err != nil {
		return 0, err // goroutine leak is possible here
	}
	ioReader.Close()
	dstFile.Close()

	return written, nil
}

// DeleteFile delete file
func DeleteFile(path string) {
	err := os.Remove(path)
	if err != nil {
		fmt.Printf("Error: %v\n\n", err)
	}
	fmt.Printf("Output: File %s deleted. \n", path)
}

// DatabaseDump dump database via mysqldump, use ./dbdump --help
func DatabaseDump() {
	// Set the command line arguments
	var (
		mysqlUser   = flag.String("u", "test", "Name of your database user.")
		mysqlHost   = flag.String("h", "localhost", "Name of your Mysql hostname.")
		mysqlDb     = flag.String("db", "test", "Database name.")
		allDatabase = flag.Bool("db-all", false, "If set dump all Mysql databases.")
		backupDir   = flag.String("backup-dir", "/opt/backup/db", "Backup directory for dumps.")
		storageDir  = flag.String("storage-dir", "/mnt/backup", "Remote storage directory for dumps.")
		gzipEnable  = flag.Bool("gzip", true, "If set gzip compression enabled.")
	)

	// Get command line arguments
	flag.Parse()
	flag.Args()

	// Get the hostname
	hostname, err := os.Hostname()

	filename := ""
	current := time.Now()
	now := fmt.Sprintf(current.Format("20060102.150405"))

	// Set Filename
	if *allDatabase {
		fmt.Printf("Output: Dumping %s databases's start...\n", hostname)
		filename = fmt.Sprintf("%s_%s.sql", hostname, now)
	} else {
		fmt.Printf("Output: Dumping database %s start...\n", *mysqlDb)
		filename = fmt.Sprintf("%s_%s.sql", *mysqlDb, now)
	}

	if *gzipEnable {
		filename += ".gz"
	}

	// Define local tmp file
	localTmpFile := fmt.Sprintf("%s/%s", *backupDir, filename)

	// Compose mysqldump command
	mysqldumpCommand := fmt.Sprintf("mysqldump -u%s -h%s --single-transaction ", *mysqlUser, *mysqlHost)
	if *allDatabase {
		mysqldumpCommand += "--all-databases "
	} else if *mysqlDb != "" {
		mysqldumpCommand += *mysqlDb
	} else {
		fmt.Println("You must specify a database name")
	}

	// TODO: Refactor to ExecCmd func or similar
	// Create database dump and store it on local tmp file
	cmd := exec.Command("/bin/bash", "-c", mysqldumpCommand)
	var out bytes.Buffer
	cmd.Stdout = &out
	err = cmd.Run()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

	// Create a gzip file of the dbdump output stream
	if *gzipEnable {
		var outGzip bytes.Buffer
		w := gzip.NewWriter(&outGzip)
		w.Write(out.Bytes())
		w.Close()

		out = outGzip
	}

	// Write the gzip stream to a tmp file
	ioutil.WriteFile(localTmpFile, out.Bytes(), 0666)
	fmt.Printf("Output: Gzip file %s created\n", localTmpFile)

	// TODO: Better to use semicolon for rm {} \;
	// Rotate dumps then synchronize it via rsync
	RunCommand("bash", "-c", fmt.Sprintf("find %s/ -name '*.sql.gz' -type f -mtime +14 -exec rm {} +", *backupDir))

	// Synchronize backup directory with storage directory
	RunCommand("bash", "-c", fmt.Sprintf("rsync -avpze --progress --stats --delete %s/ %s/", *backupDir, *storageDir))

	fmt.Printf("Output: Dump database %s finished.\n", *mysqlDb)
}

// DatabaseImport import database dump, use ./dbimport --help
func DatabaseImport() {

	// Set the command line arguments
	var (
		mysqlUser     = flag.String("user", "", "Name of your database user.")
		mysqlPassword = flag.String("password", "", "Name of your database user password.")
		mysqlHostname = flag.String("hostname", "localhost", "Name of your database hostname.")
		mysqlPort     = flag.String("port", "3306", "Name of your database port.")
		mysqlDatabase = flag.String("database", "", "Name of your database.")
	)

	c, _ := LoadConfiguration(configDir)

	// Pretty JSON configuration
	//b, err := json.MarshalIndent(c, "", "  ")
	//if err != nil {
	//	fmt.Println("Error:", err)
	//}
	//os.Stdout.Write(b)
	//fmt.Printf("\n\n")

	// Main variables
	dbName := fmt.Sprintf("%s", PassArguments())

	// Use Format for extracted file, so they don't conflicted
	current := time.Now()
	dumpFileFormat := fmt.Sprintf(current.Format("20060102.150405"))
	dumpFileDst := fmt.Sprintf("%s/dump_%s.sql", c.DatabaseDir, dumpFileFormat)

	if DirectoryExists(c.StorageDir) {
		os.Chdir(c.StorageDir)

		// Pipeline commands
		ls := exec.Command("find", ".", "-name", "*.sql.gz")
		tail := exec.Command("tail", "-1")

		// Run the pipeline
		output, stderr, err := PipeLine(ls, tail)

		if err != nil {
			fmt.Printf("Error: %s", err)
		}

		// Print the stdout, if any
		//if len(output) > 0 {
		//	fmt.Printf("Output: %s\n", output)
		//}

		// Print the stderr, if any
		if len(stderr) > 0 {
			fmt.Printf("%q: (stderr)\n", stderr)
		}

		// Convert byte output to string
		dumpFileStr := string(output[:])
		dumpFileSrc := strings.TrimSpace(dumpFileStr)

		// Copy last database dbdump to dbDir
		RunCommand("rsync", "-P", "-t", dumpFileSrc, c.DatabaseDir)

		os.Chdir(c.DatabaseDir)

		// Extract database dbdump, use time() for each extracted file *.sql
		UnpackGzipFile(dumpFileSrc, dumpFileDst)

		// Prepare database
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
			fmt.Println(err.Error())
		} else {
			fmt.Printf("\n\nSuccessfully connected to MySQL!\n\n")
		}

		drop, err := db.Exec(fmt.Sprintf("DROP DATABASE IF EXISTS %s;", dbName))
		if err != nil {
			fmt.Println("Query error:", err.Error())
		} else {
			count, _ := drop.RowsAffected()
			fmt.Printf("MySQL: Running: DROP DATABASE IF EXISTS %s;\n", dbName)
			fmt.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		create, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;", dbName))
		if err != nil {
			fmt.Println("Query error:", err.Error())
		} else {
			count, _ := create.RowsAffected()
			fmt.Printf("MySQL: Running: CREATE DATABASE %s CHARACTER SET utf8 collate utf8_unicode_ci;\n", dbName)
			fmt.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		grant, err := db.Exec(fmt.Sprintf("GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';", dbName, dbName, dbName))
		if err != nil {
			fmt.Println("Query error:", err.Error())
		} else {
			count, _ := grant.RowsAffected()
			fmt.Printf("MySQL: Running: GRANT ALL PRIVILEGES ON %s.* TO '%s'@'localhost' IDENTIFIED BY '%s';\n", dbName, dbName, dbName)
			fmt.Printf("MySQL: Query OK, %d rows affected\n\n", count)
		}

		// Import database dbdump
		RunCommand("bash", "-c", fmt.Sprintf("time mysql -u%s %s < %s", *mysqlUser, dbName, dumpFileDst))

		// Delete database dbdump's
		DeleteFile(dumpFileSrc)
		DeleteFile(dumpFileDst)

	} else {
		fmt.Printf("Error: No such file or directory %v\n", c.StorageDir)
		os.Exit(1)
	}

}

func Prepare() {
	// Set the command line arguments
	var (
		refSlug   = flag.String("CI_COMMIT_REF_SLUG", "", "Lowercased, shortened to 63 bytes, and with everything except 0-9 and a-z replaced with -. No leading / trailing -. Use in URLs, host names and domain names.")
		refName   = flag.String("CI_COMMIT_REF_NAME", "", "The branch or tag name for which project is built.")
		commitSha = flag.String("CI_COMMIT_SHA", "", "The commit revision for which project is built.")
	)

	c, _ := LoadConfiguration(configDir)

	// Get command line arguments
	flag.Parse()
	flag.Args()

	hostDir := c.RootDir + *refSlug
	settings := hostDir + c.Testing.SettingsFile
	dbconn := hostDir + c.Testing.DBConnFile

	if DirectoryExists(hostDir) {

		fmt.Printf("Directory %s exists.\n\n", hostDir)

		os.Chdir(hostDir)
		RunCommand("bash", "-c", fmt.Sprintf("git fetch --prune origin"))
		RunCommand("bash", "-c", fmt.Sprintf("git reset --hard HEAD"))

	} else {

		fmt.Printf("Create directory %s.\n\n", hostDir)
		os.Mkdir(fmt.Sprintf("%s", hostDir), 0750)
		os.Chdir(hostDir)

		RunCommand("bash", "-c", fmt.Sprintf("git init"))
		RunCommand("bash", "-c", fmt.Sprintf("git remote add -t %s -f origin %s", *refSlug, gitUrl))
		RunCommand("bash", "-c", fmt.Sprintf("git checkout %s", *commitSha))
		RunCommand("bash", "-c", fmt.Sprintf("composer install --no-dev --no-progress"))
		RunCommand("bash", "-c", fmt.Sprintf("yarn clean"))
		RunCommand("bash", "-c", fmt.Sprintf("yarn install --no-progress"))
		RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower install"))
		RunCommand("bash", "-c", fmt.Sprintf("./node_modules/.bin/bower prune"))
		RunCommand("bash", "-c", fmt.Sprintf("yarn build"))
	}

	if DirectoryExists(settings) || DirectoryExists(dbconn) {
	} else {

		fmt.Println("Run parse settings...")

		RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", settings, settings))
		RunCommand("bash", "-c", fmt.Sprintf("cp %s.example %s", dbconn, dbconn))
		RunCommand("bash", "-c", fmt.Sprintf("php -f %s %s %s %s", c.Testing.ParseSettings, hostDir, *refName, *refName))

		fmt.Println("Parse complete.")
	}
}
