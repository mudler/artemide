package main

import (
	"os"
	"strconv"

	evbus "github.com/asaskevich/EventBus"
	. "github.com/mattn/go-getopt"
	log "github.com/spf13/jwalterweatherman"

	config "github.com/mudler/artemide/pkg/config"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"

	_ "github.com/mudler/artemide/plugin/recipe/docker"
	_ "github.com/mudler/artemide/plugin/recipe/script"
)

func main() {
	if os.Getenv("DEBUG") == strconv.Itoa(1) {
		log.SetStdoutThreshold(log.LevelDebug)
	} else {
		log.SetStdoutThreshold(log.LevelInfo)
	}
	log.INFO.Println("== Artemide - the docker building system ==")
	log.INFO.Println("Engines starting")
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
		log.DEBUG.Println("Registering", i, "hook to eventbus")
		plugin.Hooks[i].Register(bus, context)
	}

	for i := range plugin.Recipes {
		log.DEBUG.Println("Registering", i, "recipe to eventbus")
		plugin.Recipes[i].Register(bus, context)
	}

	// Starting the bus show!
	bus.Publish("artemide:start") // Emitting artemide:start event thru Recipes and Hooks.

	// unpack mode.
	if unpackImage != "" && outputDir != "" {
		// Unpack mode, just unpack the image and exits.
		log.INFO.Println("Unpack mode. Unpacking", unpackImage, "to", outputDir)
		bus.Publish("artemide:source:docker", unpackImage, outputDir)
		os.Exit(0)
	}

	// Halting if no configuration file is supplied
	if configurationFile == "" {
		log.ERROR.Fatalln("I can't work without a configuration file")
	}

	configuration, _ := config.LoadConfig(configurationFile)

	log.DEBUG.Printf("%v\n", configuration)

	//bus.Publish("artemide:source:"+configuration.Source.Type, configuration.Source.Image)

	for artifactName, artifact := range configuration.Artifacts {
		log.DEBUG.Printf("Artifact: %s \n", artifactName)
		for recipeName, recipe := range artifact.Recipe {
			log.DEBUG.Printf("Signaling -> Recipe %s <- to bus\n", recipeName)
			bus.Publish("artemide:artifact:recipe:" + recipeName)
			for eventsName, event := range recipe {
				log.DEBUG.Printf("Signaling -> Event %s : (%s.%s)\n", eventsName, event.Name, event.Action)
				//bus.Publish("artemide:artifact:recipe:"+recipeName+":event:"+event.Name, event.Action)
			}
		}
	}
	//bus.Publish("artemide:recipe:type", recipe_type)

}
