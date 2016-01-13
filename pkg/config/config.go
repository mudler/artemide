package config

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"

	log "github.com/spf13/jwalterweatherman"

	"github.com/BurntSushi/toml"
)

type tomlConfig struct {
	Env          []string
	VendorString string
	Source       source   `toml:"source"`
	Artifacts    artifact `toml:"artifact"`
}

type source struct {
	Name string
	Org  string `toml:"organization"`
	Bio  string
	DOB  time.Time
}

type database struct {
	Server  string
	Ports   []int
	ConnMax int `toml:"connection_max"`
	Enabled bool
}

type server struct {
	IP string
	DC string
}

type artifact struct {
	Events map[string]server
}

func LoadConfig(f string) (tomlConfig, error) {

	filename, _ := filepath.Abs(f)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var config tomlConfig
	if _, err := toml.DecodeFile(f, &config); err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf("Title: %s\n", config.Title)
	fmt.Printf("Owner: %s (%s, %s), Born: %s\n",
		config.Owner.Name, config.Owner.Org, config.Owner.Bio,
		config.Owner.DOB)
	fmt.Printf("Database: %s %v (Max conn. %d), Enabled? %v\n",
		config.DB.Server, config.DB.Ports, config.DB.ConnMax,
		config.DB.Enabled)
	for serverName, server := range config.Servers {
		fmt.Printf("Server: %s (%s, %s)\n", serverName, server.IP, server.DC)
	}
	fmt.Printf("Client data: %v\n", config.Clients.Data)
	fmt.Printf("Client hosts: %v\n", config.Clients.Hosts)

	if config.DockerImage == "" {
		log.ERROR.Println("You need to specify a Docker image 'docker_image'")
	}
	if config.LogDir == "" {
		log.ERROR.Println("You need to specify a Log directory 'log_dir'")
	}
	log.INFO.Printf("GIT Repository: %#v\n", config.Repository)
	log.INFO.Printf("Docker Image: %#v\n", config.DockerImage)
	log.INFO.Printf("Artifacts directory: %#v\n", config.Artifacts)
	log.INFO.Printf("Separate Artifacts by commit: %#v\n", config.SeparateArtifacts)

	log.INFO.Printf("PreProcessor: %#v\n", config.PreProcessor)
	log.INFO.Printf("Log Directory: %#v\n", config.LogDir)
	log.INFO.Printf("Log Permissions: %#v\n", config.LogPerm)
	log.INFO.Printf("Poll Time: %#v\n", config.PollTime)

	return config, err
}
