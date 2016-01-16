package main

import (
	"fmt"
	"os"

	evbus "github.com/asaskevich/EventBus"
	. "github.com/mattn/go-getopt"
	jww "github.com/spf13/jwalterweatherman"

	config "github.com/mudler/artemide/pkg/config"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"

	_ "github.com/mudler/artemide/plugin/hook/script"
	_ "github.com/mudler/artemide/plugin/recipe/docker"
)

func main() {
	jww.SetStdoutThreshold(jww.LevelDebug)
	jww.INFO.Println("== Artemide - the docker building system ==")
	jww.INFO.Println("Engines starting")
	var c int
	var configurationFile string
	var unpackImage string
	var context *context.Context
	var outputDir string

	bus := evbus.New()
	OptErr = 0
	for {
		if c = Getopt("o:u:c:h"); c == EOF {
			break
		}
		switch c {
		case 'u':
			unpackImage = OptArg
		case 'o':
			outputDir = OptArg
		case 'c':
			configurationFile = OptArg
		case 'h':
			println("usage: " + os.Args[0] + " [-c config.toml -h]")
			os.Exit(1)
		}
	}

	// Register hooks and recipes to the eventbus
	for i := range plugin.Hooks {
		jww.DEBUG.Println("Registering", i, "hook to eventbus")
		plugin.Hooks[i].Register(bus, context)
	}

	for i := range plugin.Recipes {
		jww.DEBUG.Println("Registering", i, "recipe to eventbus")
		plugin.Recipes[i].Register(bus, context)
	}

	// unpack mode.
	if unpackImage != "" && outputDir != "" {
		// Unpack mode, just unpack the image and exits.
		jww.INFO.Println("Unpack mode. Unpacking", unpackImage, "to", outputDir)
		bus.Publish("artemide:source:docker", unpackImage, outputDir)
		os.Exit(0)
	}

	// Halting if no configuration file is supplied
	if configurationFile == "" {
		fmt.Println("I can't work without a configuration file")
		os.Exit(1)
	}

	configuration, _ := config.LoadConfig(configurationFile)

	jww.INFO.Printf("%v\n", configuration)

	// Starting the bus show!
	bus.Publish("artemide:start") // Emitting artemide:start event thru Recipes and Hooks.

	bus.Publish("artemide:source:"+configuration.Source.Type, configuration.Source.Image)

	for artifactName, artifact := range configuration.Artifacts {
		jww.INFO.Printf("Artifact: %s \n", artifactName)
		for _, recipe := range artifact.Recipe {
			bus.Publish("artemide:artifact:recipe:" + recipe)
		}

	}
	//bus.Publish("artemide:recipe:type", recipe_type)

}
