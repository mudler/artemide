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

	var c int
	var configurationFile string
	OptErr = 0
	for {
		if c = Getopt("c:h"); c == EOF {
			break
		}
		switch c {
		case 'c':
			configurationFile = OptArg
		case 'h':
			println("usage: " + os.Args[0] + " [-c config.toml -h]")
			os.Exit(1)
		}
	}

	if configurationFile == "1" {
		fmt.Println("I can't work without a configuration file")
		os.Exit(1)
	}

	configuration, _ := config.LoadConfig(configurationFile)

	jww.INFO.Printf("%v\n", configuration)

	var context *context.Context

	jww.INFO.Println("== Artemide - the docker building system ==")
	jww.INFO.Println("Engines starting")

	bus := evbus.New()

	for i := range plugin.Hooks {
		jww.DEBUG.Println("Registering", i, "hook to eventbus")
		plugin.Hooks[i].Register(bus, context)
	}

	for i := range plugin.Recipes {
		jww.DEBUG.Println("Registering", i, "recipe to eventbus")
		plugin.Recipes[i].Register(bus, context)
	}

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
