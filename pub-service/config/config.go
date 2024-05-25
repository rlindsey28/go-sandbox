package config

import (
	"embed"
	"log"

	"gopkg.in/yaml.v3"
)

type AppConfig struct {
	Service struct {
		Name string `yaml:"name"`
		Port string `yaml:"port"`
	}
}

//go:embed app-config.yml
var content embed.FS

func Initialize() *AppConfig {
	appConfig := &AppConfig{}
	file, err := content.ReadFile("app-config.yml")
	if err != nil {
		log.Fatalf("failed to open config file: %v", err)
	}

	err = yaml.Unmarshal(file, appConfig)
	if err != nil {
		log.Fatalf("failed to decode config file: %v", err)
	}

	return appConfig
}
