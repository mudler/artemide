package docker

import (
	"fmt"
	"os"

	evbus "github.com/asaskevich/EventBus"
	"github.com/fsouza/go-dockerclient"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
	jww "github.com/spf13/jwalterweatherman"
)

// Script construct the container arguments from the boson file
type Docker struct{}

// Process builds a list of packages from the boson file
func (d *Docker) Register(bus *evbus.EventBus, context *context.Context) { //returns args and volumes to mount
	bus.Subscribe("artemide:start", Start) //Subscribing to artemide:start, Hello will be called
}

func ExportRootfs(image string, filename string) (bool, error) {
	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)
	var err error
	var containerconfig *docker.Config
	// Container configuration. clean and simple
	containerconfig = &docker.Config{
		Image: image,
		Cmd:   []string{"true"},
	}
	// Pulling the image
	if err := client.PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", image, err)
		return false, err
	} else {
		jww.INFO.Println("Image %v pulled correctly", image)
	}

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: containerconfig,
	})
	defer client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})

	target := fmt.Sprintf("%s.tar", filename)
	writer, err := os.Create(target)
	if err != nil {
		return false, err
	}
	defer writer.Close()

	err = client.DownloadFromContainer(container.ID, docker.DownloadFromContainerOptions{Path: "/", OutputStream: writer})
	if err != nil {
		return false, err
	}

	writer.Sync()

	return false, err
}

func Start() {
	jww.INFO.Printf("Docker recipe is available")
}

func init() {
	plugin.RegisterRecipe(&Docker{})
}
