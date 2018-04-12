package config

import (
	"fmt"
	"io/ioutil"
	"log"
)

type FpmConfig struct {
	Section string
	Params  map[string]string
}

func (f *FpmConfig) Write(path string) {
	config := fmt.Sprintf("[%s]\n", f.Section)

	for k, v := range f.Params {
		config += fmt.Sprintf("%s=%s\n", k, v)
	}

	err := ioutil.WriteFile(path, []byte(config), 0644)
	if err != nil {
		log.Fatalln("WriteFile:", err)
	}
}
