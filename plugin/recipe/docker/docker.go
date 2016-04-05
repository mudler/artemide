package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/fsouza/go-dockerclient"

	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
	jww "github.com/spf13/jwalterweatherman"
)

const SEPARATOR = string(filepath.Separator)
const ROOT_FS = "." + SEPARATOR + "rootfs_overlay"

// Script construct the container arguments from the boson file
type Docker struct{}

// Process builds a list of packages from the boson file
func (d *Docker) Register(bus *evbus.EventBus, context *context.Context) { //returns args and volumes to mount

	client, _ := NewClient("unix:///var/run/docker.sock")

	bus.Subscribe("artemide:start", Start) //Subscribing to artemide:start, Hello will be called
	bus.Subscribe("artemide:source:docker", client.Unpack)

}

type Client struct {
	docker *docker.Client
}

func NewClient(endpoint string) (*Client, error) {
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, err
	}
	return &Client{
		docker: client,
	}, nil
}

func (client *Client) Unpack(image string, dirname string) (bool, error) {
	var err error

	if dirname == "" {
		dirname = ROOT_FS
	}

	os.MkdirAll(dirname, 0777)

	filename, err := ioutil.TempFile(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary file")
	}
	os.Remove(filename.Name())

	// Pulling the image
	jww.INFO.Printf("Pulling the docker image %s\n", image)
	if err := client.docker.PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", image, err)
		return false, err
	} else {
		jww.INFO.Println("Image", image, "pulled correctly")
	}

	History, _ := client.docker.ImageHistory(image)

	for i := len(History) - 1; i >= 0; i-- {
		layer := History[i]
		layerCreated := time.Unix(layer.Created, 0)
		jww.DEBUG.Println("Layer ", layer.ID, layerCreated)

	}

	jww.INFO.Println("Creating container")
	//flatten.Flatten("flat", image)

	container, err := client.docker.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: image,
			Cmd:   []string{"true"},
		},
	})
	defer func(*docker.Container) {
		client.docker.RemoveContainer(docker.RemoveContainerOptions{
			ID:    container.ID,
			Force: true,
		})
	}(container)

	//squashImage(container.ID, "artemide")

	target := fmt.Sprintf("%s.tar", filename.Name())
	jww.DEBUG.Printf("Writing to target %s\n", target)
	writer, err := os.Create(target)
	if err != nil {
		return false, err
	}

	err = client.docker.ExportContainer(docker.ExportContainerOptions{ID: container.ID, OutputStream: writer})
	if err != nil {
		jww.FATAL.Fatalln("Couldn't export container, sorry", err)
	}

	writer.Sync()

	writer.Close()
	jww.INFO.Println("Extracting to", dirname)

	untar(target, dirname)
	err = os.Remove(target)
	if err != nil {
		jww.ERROR.Println("could not remove temporary file", target)
	}
	prepareRootfs(dirname)

	return true, err
}

func prepareRootfs(dirname string) {

	err := os.Remove(dirname + SEPARATOR + ".dockerenv")
	if err != nil {
		jww.ERROR.Println("could not remove docker env file")
	}

	err = os.Remove(dirname + SEPARATOR + ".dockerinit")
	if err != nil {
		jww.ERROR.Println("could not remove docker init file")
	}

	err = os.MkdirAll(dirname+SEPARATOR+"dev", 0751)
	if err != nil {
		jww.ERROR.Println("could not create dev folder")
	}

	// Google DNS as default
	d1 := []byte("nameserver 8.8.8.8\nnameserver 8.8.4.4\n")
	err = ioutil.WriteFile(dirname+SEPARATOR+"etc"+SEPARATOR+"resolv.conf", d1, 0644)

}

func Start() {
	jww.DEBUG.Printf("[recipe] Docker is available")
}

func squashImage(container string, newimage string) string {
	cmd := "docker export " + container + "| docker import - " + newimage
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		jww.FATAL.Fatalf("Failed to execute command: %s", cmd)
	}
	return string(out)
}
func untar(src string, dst string) string {

	// this should be used instead https://github.com/yuuki1/droot/blob/d0a19947ca0ab057d1eb8cfd471ce6863675b64f/archive/util.go#L19
	// temporary code to move on.
	cmd := "tar -xf " + src + " -C " + dst + " --exclude='dev'"
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		jww.FATAL.Fatalf("Failed to execute command: %s", cmd)
	}
	return string(out)
}

func init() {
	plugin.RegisterRecipe(&Docker{})
}
