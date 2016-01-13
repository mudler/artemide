// Package plugin register plugins here, the registry keep tracks of plugins to redirect the messages
package plugin

import (
	"reflect"
	"strings"

	evbus "github.com/asaskevich/EventBus"
	"github.com/mudler/artemide/pkg/context"
)

// Hook register it's events to the eventbus
type Hook interface {
	Register(*evbus.EventBus, *context.Context) // processor gets the workdir and the config file
}

// Recipe is a special type of Hook
type Recipe interface {
	Hook
}

// Hooks contains a map of Hook
var Hooks = map[string]Hook{}

// Recipes contains a map of Recipe
var Recipes = map[string]Recipe{}

// RegisterHook Registers a Hook
func RegisterHook(h Hook) {
	Hooks[keyOf(h)] = h
}

// RegisterRecipe Registers a Recipe
func RegisterRecipe(r Recipe) {
	Recipes[keyOf(r)] = r
}

func keyOf(h Hook) string {
	return strings.TrimPrefix(reflect.TypeOf(h).String(), "*")
}
