package docker

import (
	evbus "github.com/asaskevich/EventBus"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
	jww "github.com/spf13/jwalterweatherman"
)

// Script construct the container arguments from the boson file
type Docker struct{}

// Process builds a list of packages from the boson file
func (d *Docker) Register(bus *evbus.EventBus, context *context.Context) { //returns args and volumes to mount
	bus.Subscribe("artemide:start", Hello) //Subscribing to artemide:start, Hello will be called
}

func Hello() {
	jww.INFO.Printf("Helloooo!!! it is WORKING!")
}

func init() {
	plugin.RegisterRecipe(&Docker{})
}
