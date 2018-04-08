package archive

import (
	"os"
	"log"
	"compress/gzip"
	"archive/tar"
	"io"
)

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
