package main

import (
	evbus "github.com/asaskevich/EventBus"
	jww "github.com/spf13/jwalterweatherman"

	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
	_ "github.com/mudler/artemide/plugin/hook/script"
	_ "github.com/mudler/artemide/plugin/recipe/docker"
)

func main() {

	var context *context.Context

	jww.SetStdoutThreshold(jww.LevelInfo)

	jww.INFO.Printf("== Artemide - the docker building system ==")
	jww.INFO.Printf("Engines starting")

	bus := evbus.New()

	for i := range plugin.Hooks {
		jww.DEBUG.Printf("Registering hooks to eventbus")
		plugin.Hooks[i].Register(bus, context)
	}

	for i := range plugin.Recipes {
		jww.DEBUG.Printf("Registering recipes to eventbus")
		plugin.Recipes[i].Register(bus, context)
	}

	bus.Publish("artemide:start") // Emitting artemide:start event thru Recipes and Hooks.

	//bus.Publish("artemide:source:docker", docker_image)
	//bus.Publish("artemide:recipe:type", recipe_type)

}
