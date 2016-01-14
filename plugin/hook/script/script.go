package script

import (
	evbus "github.com/asaskevich/EventBus"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
)

// Script construct the container arguments from the boson file
type Script struct{}

// Process builds a list of packages from the boson file
func (s *Script) Register(bus *evbus.EventBus, context *context.Context) { //returns args and volumes to mount
	bus.Subscribe("artemide:start", Hello) //Subscribing to artemide:start, Hello will be called
}

func Hello() {
}

func init() {
	plugin.RegisterHook(&Script{})
}
