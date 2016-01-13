package config

import (
	"io/ioutil"
	"path/filepath"
	"regexp"

	log "github.com/spf13/jwalterweatherman"

	"gopkg.in/yaml.v2"
)

// Config represent the yaml configuration file
type Config struct {
	// Firewall_network_rules map[string]Options `yaml:"repository"`
	Repository            string `yaml:"repository"`
	RepositoryStripped    string
	DockerImage           string              `yaml:"docker_image"`
	DockerSkipPull        bool                `yaml:"docker_skip_pull"`
	DockerCommit          bool                `yaml:"docker_commit"`
	Commit                map[string]string   `yaml:"commit"`
	DockerImageEntrypoint []string            `yaml:"docker_image_entrypoint"`
	PreProcessor          string              `yaml:"preprocessor"`
	Provisioner           map[string][]string `yaml:"provisioner"`
	PostProcessor         []string            `yaml:"postprocessors"`
	PollTime              int                 `yaml:"polltime"`
	Artifacts             string              `yaml:"artifacts_dir"`
	SeparateArtifacts     bool                `yaml:"separate_artifacts"`
	LogDir                string              `yaml:"log_dir"`
	LogPerm               int                 `yaml:"logfile_perm"`
	Env                   []string            `yaml:"env"`
	Args                  []string            `yaml:"args"`
	TmpDir                string              `yaml:"tmpdir"`
	Volumes               []string            `yaml:"volumes"`
	WorkDir               string
}

//type Options struct {
//   Src string
//   Dst string
//}

// LoadConfig generate a Config from the given yaml file path
func LoadConfig(f string) (Config, error) {

	filename, _ := filepath.Abs(f)
	yamlFile, err := ioutil.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	var config Config
	config.SeparateArtifacts = false
	config.PollTime = 5
	config.LogPerm = int(0644)
	config.Artifacts = "/tmp"
	config.TmpDir = "/var/tmp/boson/"
	config.DockerSkipPull = false
	err = yaml.Unmarshal(yamlFile, &config)

	r, _ := regexp.Compile(`^.*?\/\/`)
	config.RepositoryStripped = r.ReplaceAllString(config.Repository, "")
	config.WorkDir = config.TmpDir + config.RepositoryStripped

	if config.DockerImage == "" {
		log.ERROR.Print("You need to specify a Docker image 'docker_image'")
	}
	if config.LogDir == "" {
		log.ERROR.Print("You need to specify a Log directory 'log_dir'")
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
