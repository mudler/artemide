package main

import (
	evbus "github.com/asaskevich/EventBus"
	plugin "github.com/mudler/artemide/plugin"
	_ "github.com/mudler/artemide/plugin/hook/script"
	jww "github.com/spf13/jwalterweatherman"
)

func main() {

	jww.SetStdoutThreshold(jww.LevelInfo)

	jww.INFO.Printf("== Artemide - the docker building system ==")
	jww.INFO.Printf("Engines starting")

	bus := evbus.New()

	for i := range plugin.Hooks {
		jww.DEBUG.Printf("Registering hooks to eventbus")
		plugin.Hooks[i].Register(bus)
	}

	for i := range plugin.Recipes {
		jww.DEBUG.Printf("Registering recipes to eventbus")
		plugin.Recipes[i].Register(bus)
	}

	bus.Publish("artemide:start") // Emitting artemide:start event thru Recipes and Hooks.

}
