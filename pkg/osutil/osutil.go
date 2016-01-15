package osutil

import (
	"io"
	"os"
	"os/exec"
	fp "path/filepath"
	"strings"
	"syscall"

	"github.com/docker/docker/pkg/mount"
	log "github.com/spf13/jwalterweatherman"
)

func ExistsFile(file string) bool {
	f, err := os.Stat(file)
	return err == nil && !f.IsDir()
}

func ExistsDir(dir string) bool {
	if f, err := os.Stat(dir); os.IsNotExist(err) || !f.IsDir() {
		return false
	}
	return true
}

func IsDirEmpty(dir string) (bool, error) {
	f, err := os.Open(dir)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, nil
}

func RunCmd(name string, arg ...string) error {
	log.DEBUG.Println("runcmd: ", name, arg)
	out, err := exec.Command(name, arg...).CombinedOutput()
	if len(out) > 0 {
		log.DEBUG.Println(string(out))
	}
	if err != nil {
		return err
	}
	return nil
}

func Cp(from, to string) error {
	if err := RunCmd("cp", "-p", from, to); err != nil {
		return err
	}
	return nil
}

func GetMountsByRoot(rootDir string) ([]*mount.Info, error) {
	mounts, err := mount.GetMounts()
	if err != nil {
		return nil, err
	}

	targets := make([]*mount.Info, 0)
	for _, m := range mounts {
		if strings.HasPrefix(m.Mountpoint, fp.Clean(rootDir)) {
			targets = append(targets, m)
		}
	}

	return targets, nil
}

func UmountRoot(rootDir string) error {
	mounts, err := GetMountsByRoot(rootDir)
	if err != nil {
		return err
	}

	for _, m := range mounts {
		if err := mount.Unmount(m.Mountpoint); err != nil {
			return err
		}
		log.DEBUG.Println("umount:", m.Mountpoint)
	}

	return nil
}

// Mknod unless path does not exists.
func Mknod(path string, mode uint32, dev int) error {
	if ExistsFile(path) {
		return nil
	}
	if err := syscall.Mknod(path, mode, dev); err != nil {
		return err
	}
	return nil
}

// Symlink, but ignore already exists file.
func Symlink(oldname, newname string) error {
	if err := os.Symlink(oldname, newname); err != nil {
		// Ignore already created symlink
		if _, ok := err.(*os.LinkError); !ok {
			return err
		}
	}
	return nil
}

func Execv(cmd string, args []string, env []string) error {
	name, err := exec.LookPath(cmd)
	if err != nil {
		return err
	}

	log.DEBUG.Println("exec: ", name, args)

	return syscall.Exec(name, args, env)
}
