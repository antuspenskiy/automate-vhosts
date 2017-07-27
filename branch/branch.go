package branch

import (
	"fmt"
	"os/exec"
	"bytes"
	"os"
	"strings"
	"io"
	"compress/gzip"
)

func IsExist(path string) bool {
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Output: File or directory exists: %s\n", path)
		return true
	}
	fmt.Printf("Error: No such file or directory: %s\n", path)
	return false
}

func ExecCmd(path string, args ...string) {
	fmt.Printf("Running: %q %q\n", path, strings.Join(args, " "))
	cmd := exec.Command(path, args...)
	bs, err := cmd.CombinedOutput()
	fmt.Printf("Output: %s", bs)
	fmt.Printf("Error: %v\n\n", err)
}

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
