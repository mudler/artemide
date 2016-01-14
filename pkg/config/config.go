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
	Type  string
	Image string
}

type event struct {
	ActionType string `toml:"action_type"` // script
	Action     string `toml:"action"`
	Name       string `toml:"name"`
}

type artifact struct {
	Recipe        []string
	Destination   string
	checksum_type []string
	Events        map[string]event `toml:"event"`
}

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
		for eventsName, event := range artifact.Events {
			log.INFO.Printf("-> Event %s : (%s %s %s)\n", eventsName, event.Name, event.Action, event.ActionType)

		}
	}

	return config, err
}
