package docker

import (
	"fmt"
	"io"
	"io/ioutil"
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
	bus.Subscribe("artemide:source:docker", exportRootfs)

}

// exportRootfs exports the rootfs of a docker image. It is the result of the squash of all the layers
func exportRootfs(image string) (bool, error) {
	var err error

	filename, err := ioutil.TempFile(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary file")
	}

	endpoint := "unix:///var/run/docker.sock"
	client, _ := docker.NewClient(endpoint)
	var containerconfig *docker.Config
	// Container configuration. clean and simple
	containerconfig = &docker.Config{
		Image: image,
		Cmd:   []string{"true"},
	}
	// Pulling the image
	jww.INFO.Printf("Pulling the docker image %s\n", image)
	if err := client.PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", image, err)
		return false, err
	} else {
		jww.INFO.Println("Image %v pulled correctly", image)
	}
	jww.INFO.Println("Creating container")

	container, err := client.CreateContainer(docker.CreateContainerOptions{
		Config: containerconfig,
	})

	// err = client.StartContainer(container.ID, &docker.HostConfig{})
	// if err != nil {
	// 	jww.ERROR.Fatal(err.Error())
	// 	return false, err
	// }
	// client.WaitContainer(container.ID)

	target := fmt.Sprintf("%s.tar", filename.Name())
	jww.INFO.Printf("Writing to target %s\n", target)

	writer, err := os.Create(target)
	if err != nil {
		return false, err
	}

	err = client.ExportContainer(docker.ExportContainerOptions{ID: container.ID, OutputStream: writer})
	if err != nil {
		return false, err
	}

	writer.Sync()
	client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
	writer.Close()

	io.Copy()
	return true, err
}

func Start() {
	jww.INFO.Printf("Docker recipe is available")
}

func init() {
	plugin.RegisterRecipe(&Docker{})
}
