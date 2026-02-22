package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Apps []*ApplicationConfig `yaml:"apps"`
}

type ApplicationConfig struct {
	Host                string            `yaml:"host"`
	Instances           []*InstanceConfig `yaml:"instances"`
	HealthUri           string            `yaml:"health_uri"`
	Timeout             string            `yaml:"timeout"`
	HealthCheckCooldown string            `yaml:"health_check_cooldown"`
	Strategy            string            `yaml:"strategy"`
}

type InstanceConfig struct {
	Url string `yaml:"url"`
}

func LoadConfig(path string) (*Config, error) {
	config := &Config{}
	file, err := os.ReadFile(path)

	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(file, config)

	if err != nil {
		return nil, err
	}

	return config, nil
}
