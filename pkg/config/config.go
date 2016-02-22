package config

import (
	"path/filepath"

	log "github.com/spf13/jwalterweatherman"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	Env          []string
	VendorString string              `toml:"vendor"`
	Source       source              `toml:"source"`
	Artifacts    map[string]artifact `toml:"artifact"`
}

type source struct {
	Type  string `toml:"type"`
	Image string `toml:"image"`
}

type event struct {
	Action string `toml:"action"`
	Name   string `toml:"name"`
}

type artifact struct {
	Destination   string
	checksum_type []string
	Recipe        map[string]events
}

type events map[string]event

func LoadConfig(f string) (tomlConfig, error) {

	filename, _ := filepath.Abs(f)
	var err error
	var config tomlConfig
	if _, err = toml.DecodeFile(filename, &config); err != nil {
		log.ERROR.Fatal(err)
		return config, err
	}

	log.INFO.Printf("Vendor: %s\n", config.VendorString)
	log.INFO.Printf("Source Type: %s\n", config.Source.Type)
	log.INFO.Printf("Source Image: %s\n", config.Source.Image)

	for artifactName, artifact := range config.Artifacts {
		log.INFO.Printf("Artifact: %s \n", artifactName)
		for recipeName, recipe := range artifact.Recipe {
			log.INFO.Printf("-> Recipe %s <-\n", recipeName)

			for eventsName, event := range recipe {
				log.INFO.Printf("-> Event %s : (%s.%s)\n", eventsName, event.Name, event.Action)
			}

		}
	}

	return config, err
}
