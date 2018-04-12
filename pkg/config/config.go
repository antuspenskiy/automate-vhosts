package config

import (
	"io"
	"log"
	"text/template"
	"bytes"
	"os"
	"strings"
	"encoding/json"

	"github.com/spf13/viper"
)

// ReadConfig read json environment file from directory
func ReadConfig(filename string) (*viper.Viper, error) {
	v := viper.New()
	v.SetConfigName(filename)
	v.AddConfigPath("/opt/scripts/config")
	v.AutomaticEnv()
	err := v.ReadInConfig()
	return v, err
}

// EncodeTo save configuration files in json
func EncodeTo(w io.Writer, i interface{}) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(i); err != nil {
		log.Fatalf("failed encoding to writer: %s", err)
	}
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
