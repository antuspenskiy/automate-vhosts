package config

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"text/template"

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

// WriteJSONToFile write json file
func WriteJSONToFile(path string, i interface{}) error {
	data, _ := json.Marshal(i)
	err := ioutil.WriteFile(path, data, 0644)
	if err != nil {
		log.Fatalf("Error: %v\n\n", err)
		return err
	}
	return nil
}

// WriteToFile write file
func WriteToFile(path string, s string) error {
	err := ioutil.WriteFile(path, []byte(s), 0644)
	if err != nil {
		log.Fatalf("Error: %v\n\n", err)
		return err
	}
	return nil
}

// PrettyJSON print json file in pretty format
func PrettyJSON(i interface{}) string {
	data, err := json.MarshalIndent(i, "", " ")
	if err != nil {
		log.Fatalln("MarshalIndent:", err)
	}
	return string(data)
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
