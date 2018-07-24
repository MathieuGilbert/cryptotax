package config

import (
	"encoding/json"
	"io/ioutil"
)

// Configuration stores the app config
type Configuration struct {
	Database struct {
		Host     string
		Port     int
		User     string
		DBName   string
		Password string
		SSLMode  string
	}
}

// Read will load the config file into the Configuration object
func Read(fileName string) (*Configuration, error) {
	file, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	c := &Configuration{}
	if err = json.Unmarshal(file, &c); err != nil {
		return nil, err
	}

	return c, nil
}
