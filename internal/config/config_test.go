package config

import (
	"os"
	"reflect"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	scenarios := []struct {
		errorReadingFile bool
		errorReadingYaml bool
		expectedConfig   *Config
	}{
		{true, false, nil},
		{false, true, nil},
		{false, false, &Config{
			Apps: []*ApplicationConfig{
				{
					"http://app1-host.com",
					[]*InstanceConfig{
						{"http://localhost:8080"},
						{"http://localhost:8081"},
					},
					"/health",
					"10s",
					"60s",
					"least_connections",
				},
				{
					"http://app2-host.com",
					[]*InstanceConfig{
						{"http://localhost:9090"},
						{"http://localhost:9091"},
					},
					"/api/v2/health",
					"5s",
					"45s",
					"",
				},
			},
		}},
	}

	for _, scenario := range scenarios {
		var pathToFile string
		var temp *os.File

		// Setup
		if scenario.errorReadingFile {
			pathToFile = "bad/path/config.yaml"
		} else {
			var err error
			temp, err = os.CreateTemp("../../", "test-config-*.yaml")

			if err != nil {
				t.Errorf("Error creating temp file: %v", err)
			}

			pathToFile = temp.Name()

			if scenario.errorReadingYaml {
				temp.WriteString("invalid yaml")
			} else {
				temp.WriteString(`
apps:
  - host: http://app1-host.com
    health_uri: /health
    timeout: 10s
    health_check_cooldown: 60s
    instances:
      - url: http://localhost:8080
      - url: http://localhost:8081
    strategy: least_connections
  - host: http://app2-host.com
    health_uri: /api/v2/health
    timeout: 5s
    health_check_cooldown: 45s
    instances:
      - url: http://localhost:9090
      - url: http://localhost:9091`)
			}
		}

		// Meat and Potatoes
		config, err := LoadConfig(pathToFile)

		if scenario.errorReadingFile && err == nil {
			t.Errorf("Should have errored reading config file: %v", pathToFile)
		}

		if scenario.errorReadingYaml && err == nil {
			t.Errorf("Should have errored unmarshalling yaml: %v", pathToFile)
		}

		if scenario.expectedConfig != nil && config == nil {
			t.Errorf("Unexpected error reading config file: %v -- %v", pathToFile, err)
		}

		if scenario.expectedConfig != nil && config != nil {
			if len(scenario.expectedConfig.Apps) != len(config.Apps) {
				t.Errorf("Expected %d apps, got %d", len(scenario.expectedConfig.Apps), len(config.Apps))
			}

			if !reflect.DeepEqual(scenario.expectedConfig.Apps[0], config.Apps[0]) {
				t.Errorf("Expected app %+v, got %+v", scenario.expectedConfig.Apps[0], config.Apps[0])
			}

			if !reflect.DeepEqual(scenario.expectedConfig.Apps[1], config.Apps[1]) {
				t.Errorf("Expected app %+v, got %+v", scenario.expectedConfig.Apps[1], config.Apps[1])
			}
		}

		// Teardown
		if !scenario.errorReadingFile {
			os.Remove(pathToFile)
			temp.Close()
		}
	}
}
