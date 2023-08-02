// package config
package config

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

func loadYaml(inputFile string, conf *SwoopConfig) error {
	readFile, err := ioutil.ReadFile(inputFile)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(readFile, conf)
	if err != nil {
		return err
	}

	return nil
}

func Parse(configFile string) (*SwoopConfig, error) {
	conf := &SwoopConfig{}

	err := loadYaml(configFile, conf)
	if err != nil {
		return nil, err
	}

	return conf, nil
}
