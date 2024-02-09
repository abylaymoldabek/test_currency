package config

import (
	"encoding/json"
	"os"
)

type DBConfig struct {
	Host     string `json:"Host"`
	Port     int    `json:"Port"`
	User     string `json:"User"`
	Password string `json:"Password"`
	DBName   string `json:"DBName"`
}

type ServerConfig struct {
	Port int `json:"Port"`
}

type Config struct {
	DBConfig     DBConfig     `json:"DBConfig"`
	ServerConfig ServerConfig `json:"ServerConfig"`
}

func LoadConfig(filename string) (Config, error) {
	var config Config

	file, err := os.Open(filename)
	if err != nil {
		return config, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return config, err
	}

	return config, nil
}
