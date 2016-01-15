package docker

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/docker/docker/pkg/archive"
	"github.com/fsouza/go-dockerclient"

	"github.com/mudler/artemide/pkg/archive"
	"github.com/mudler/artemide/pkg/context"
	plugin "github.com/mudler/artemide/plugin"
	jww "github.com/spf13/jwalterweatherman"
)

const ROOT_FS = "." + string(filepath.Separator) + "rootfs_overlay"

// Script construct the container arguments from the boson file
type Docker struct{}

// Process builds a list of packages from the boson file
func (d *Docker) Register(bus *evbus.EventBus, context *context.Context) { //returns args and volumes to mount
	bus.Subscribe("artemide:start", Start) //Subscribing to artemide:start, Hello will be called
	bus.Subscribe("artemide:source:docker", UnpackImage)

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

func UnpackImage(image string) {
	client, _ := NewClient("unix:///var/run/docker.sock")
	client.Unpack(image)
}

func (client *Client) Unpack(image string) (bool, error) {
	var err error

	os.MkdirAll(ROOT_FS, 0777)
	dirname := ROOT_FS
	//	dirname, err := filepath.Abs(filepath.Dir(ROOT_FS))
	//	if err != nil {
	//		jww.FATAL.Fatal(err)
	//	}

	filename, err := ioutil.TempFile(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary file")
	}
	os.Remove(filename.Name())
	// dirname, err := ioutil.TempDir(os.TempDir(), "artemide")
	// if err != nil {
	// 	jww.FATAL.Fatal("Couldn't create the temporary dir", dirname)
	// }

	History, _ := client.docker.ImageHistory(image)

	for i := len(History) - 1; i >= 0; i-- {
		layer := History[i]
		layerCreated := time.Unix(layer.Created, 0)
		jww.INFO.Println("Layer ", layer.ID, layerCreated)

	}

	// Pulling the image
	jww.INFO.Printf("Pulling the docker image %s\n", image)
	if err := client.docker.PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", image, err)
		return false, err
	} else {
		jww.INFO.Println("Image", image, "pulled correctly")
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

	return true, err
}

func (c *Client) ExportImage(imageID string) (io.ReadCloser, error) {
	History, _ := c.docker.ImageHistory(imageID)

	for i := len(History) - 1; i >= 0; i-- {
		layer := History[i]
		layerCreated := time.Unix(layer.Created, 0)
		jww.INFO.Println("Layer ", layer.ID, layerCreated)

	}

	var containerconfig *docker.Config
	// Container configuration. clean and simple
	containerconfig = &docker.Config{
		Image: imageID,
		Cmd:   []string{"true"},
	}
	//Pulling the image
	jww.INFO.Printf("Pulling the docker image %s\n", imageID)
	if err := c.docker.PullImage(docker.PullImageOptions{Repository: imageID}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", imageID, err)
	} else {
		jww.INFO.Println("Image", imageID, "pulled correctly")
	}
	jww.INFO.Println("Creating container")

	container, err := c.docker.CreateContainer(docker.CreateContainerOptions{
		Config: containerconfig})
	if err != nil {
		return nil, err
	}

	// target, err := ioutil.TempDir(os.TempDir(), "artemide")
	// if err != nil {
	// 	jww.FATAL.Fatalln("Couldn't create the temporary directory")
	// }
	//writer, err := os.Create(target)

	pReader, pWriter := io.Pipe()

	go func() {
		defer func() {
			c.docker.RemoveContainer(docker.RemoveContainerOptions{
				ID:    container.ID,
				Force: true,
			})
		}()
		jww.DEBUG.Println("Exporting container contents")

		err := c.docker.ExportContainer(docker.ExportContainerOptions{
			ID:           container.ID,
			OutputStream: pWriter,
		})
		if err != nil {
			pWriter.CloseWithError(err)
		} else {
			pWriter.Close()
		}
		jww.DEBUG.Println("Exporting finished")
	}()

	return pReader, err

}

func unpackImage(image string) {
	client, _ := NewClient("unix:///var/run/docker.sock")
	dirname, err := ioutil.TempDir(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary dir")
	}
	jww.INFO.Println("Unpacking image", image)

	pReader, err := client.ExportImage(image)
	if err != nil {
		jww.FATAL.Fatalln(err)
	}
	defer pReader.Close()

	gzipReader := archiveutils.Compress(pReader)
	defer gzipReader.Close()

	//defer pipe.Close()
	//
	// jww.INFO.Println("Extracting content to", dirname)
	// //	archiveutils.ExtractTar(pipe, dirname)
	archive.Untar(pReader, dirname, &archive.TarOptions{
		Compression:     archive.Gzip,
		NoLchown:        false,
		ExcludePatterns: []string{"dev/"}, // prevent operation not permitted
	})
}

// exportRootfs exports the rootfs of a docker image. It is the result of the squash of all the layers
func exportRootfs(image string) (bool, error) {
	var err error

	filename, err := ioutil.TempFile(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary file")
	}
	dirname, err := ioutil.TempDir(os.TempDir(), "artemide")
	if err != nil {
		jww.FATAL.Fatal("Couldn't create the temporary dir", dirname)
	}

	client, _ := NewClient("unix:///var/run/docker.sock")

	History, _ := client.docker.ImageHistory(image)

	for i := len(History) - 1; i >= 0; i-- {
		layer := History[i]
		layerCreated := time.Unix(layer.Created, 0)
		jww.INFO.Println("Layer ", layer.ID, layerCreated)

	}

	// Pulling the image
	jww.INFO.Printf("Pulling the docker image %s\n", image)
	if err := client.docker.PullImage(docker.PullImageOptions{Repository: image}, docker.AuthConfiguration{}); err != nil {
		jww.ERROR.Printf("error pulling %s image: %s\n", image, err)
		return false, err
	} else {
		jww.INFO.Println("Image %v pulled correctly", image)
	}
	jww.INFO.Println("Creating container")
	//flatten.Flatten("flat", image)

	container, err := client.docker.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: image,
			Cmd:   []string{"true"},
		},
	})

	//squashImage(container.ID, "artemide")

	container, err = client.docker.CreateContainer(docker.CreateContainerOptions{
		Config: &docker.Config{
			Image: image,
			Cmd:   []string{"true"},
		},
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

	err = client.docker.ExportContainer(docker.ExportContainerOptions{ID: container.ID, OutputStream: writer})
	if err != nil {
		return false, err
	}

	writer.Sync()
	client.docker.RemoveContainer(docker.RemoveContainerOptions{
		ID:    container.ID,
		Force: true,
	})
	writer.Close()

	return true, err
}

func Start() {
	jww.INFO.Printf("Docker recipe is available")
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
	cmd := "tar -xf " + src + " -C " + dst
	out, err := exec.Command("bash", "-c", cmd).Output()
	if err != nil {
		jww.FATAL.Fatalf("Failed to execute command: %s", cmd)
	}
	return string(out)
}

func init() {
	plugin.RegisterRecipe(&Docker{})
}
