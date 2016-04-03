package osutil

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	fp "path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/mount"
	"github.com/docker/libcontainer/system"
	"github.com/fsouza/go-dockerclient/external/github.com/docker/docker/pkg/fileutils"
	log "github.com/spf13/jwalterweatherman"
	"github.com/yuuki1/go-group"
	"golang.org/x/sys/unix"
)

const (
	mountinfoFormat = "%d %d %d:%d %s %s %s %s"
)

func Setgid(id int) error {
	return system.Setgid(id)
}

func Setuid(id int) error {
	return system.Setuid(id)
}
func bindMount(bindDir string, rootDir string, readonly bool) error {
	var srcDir, destDir string

	d := strings.SplitN(bindDir, ":", 2)
	if len(d) < 2 {
		srcDir = d[0]
	} else {
		srcDir, destDir = d[0], d[1]
	}
	if destDir == "" {
		destDir = srcDir
	}

	ok, err := IsDirEmpty(srcDir)
	if err != nil {
		return err
	}

	containerDir := fp.Join(rootDir, destDir)

	if err := fileutils.CreateIfNotExists(containerDir, true); err != nil { // mkdir -p
		log.FATAL.Fatalln("Failed to create directory:", containerDir, err)
	}

	ok, err = IsDirEmpty(containerDir)
	if err != nil {
		return err
	}
	if ok {
		log.DEBUG.Println("bind mount", bindDir, "to", containerDir)
		if err := mount.Mount(srcDir, containerDir, "none", "bind,rw"); err != nil {
			log.FATAL.Fatalln("Failed to bind mount:", containerDir, err)
		}

		if readonly {
			log.DEBUG.Println("robind mount", bindDir, "to", containerDir)
			if err := mount.Mount(srcDir, containerDir, "none", "remount,ro,bind"); err != nil {
				log.FATAL.Fatalln("Failed to robind mount:", containerDir, err)
			}
		}
	}

	return nil
}

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

func createDevices(rootDir string, uid, gid int) error {
	nullDir := fp.Join(rootDir, os.DevNull)
	if err := Mknod(nullDir, syscall.S_IFCHR|uint32(os.FileMode(0666)), 1*256+3); err != nil {
		return err
	}

	if err := os.Lchown(nullDir, uid, gid); err != nil {
		log.FATAL.Fatalln("Failed to lchown:", nullDir, err)
	}

	zeroDir := fp.Join(rootDir, "/dev/zero")
	if err := Mknod(zeroDir, syscall.S_IFCHR|uint32(os.FileMode(0666)), 1*256+3); err != nil {
		return err
	}

	if err := os.Lchown(zeroDir, uid, gid); err != nil {
		log.FATAL.Fatalln("Failed to lchown:", zeroDir, err)
	}

	for _, f := range []string{"/dev/random", "/dev/urandom"} {
		randomDir := fp.Join(rootDir, f)
		if err := Mknod(randomDir, syscall.S_IFCHR|uint32(os.FileMode(0666)), 1*256+9); err != nil {
			return err
		}

		if err := os.Lchown(randomDir, uid, gid); err != nil {
			log.FATAL.Fatalln("Failed to lchown:", randomDir, err)
		}
	}

	return nil
}

// Unmount will unmount the target filesystem, so long as it is mounted.
func Unmount(target string, flag int) error {
	if mounted, err := Mounted(target); err != nil || !mounted {
		return err
	}
	return ForceUnmount(target, flag)
}

// ForceUnmount will force an unmount of the target filesystem, regardless if
// it is mounted or not.
func ForceUnmount(target string, flag int) (err error) {
	// Simple retry logic for unmount
	for i := 0; i < 10; i++ {
		if err = syscall.Unmount(target, flag); err == nil {
			return err
		}
		time.Sleep(100 * time.Millisecond)
	}
	return
}

func Mounted(mountpoint string) (bool, error) {
	mntpoint, err := os.Stat(mountpoint)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}
	parent, err := os.Stat(fp.Join(mountpoint, ".."))
	if err != nil {
		return false, err
	}
	mntpointSt := mntpoint.Sys().(*syscall.Stat_t)
	parentSt := parent.Sys().(*syscall.Stat_t)
	return mntpointSt.Dev != parentSt.Dev, nil
}

func GetMountsByRoot(rootDir string) ([]string, error) {
	f, err := os.Open("/proc/self/mountinfo")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	s := bufio.NewScanner(f)
	mountpoints := make([]string, 0)
	re := regexp.MustCompile(fmt.Sprintf("^%s", rootDir))

	for s.Scan() {
		if err := s.Err(); err != nil {
			return nil, err
		}

		var (
			text           = s.Text()
			mountpoint     string
			d1, d2, d3, d4 int
			s1, s2, s3     string
		)

		if _, err := fmt.Sscanf(text, mountinfoFormat, &d1, &d2,
			&d3, &d4, &s1, &mountpoint, &s2, &s3); err != nil {
			return nil, fmt.Errorf("Scanning '%s' failed: %s", text, err)
		}

		if !re.MatchString(mountpoint) {
			continue
		}

		mountpoints = append(mountpoints, mountpoint)
	}
	return mountpoints, nil
}

func UmountRoot(rootDir string) (err error) {
	mounts, err := GetMountsByRoot(rootDir)
	if err != nil {
		return err
	}

	for _, mount := range mounts {
		if err = Unmount(mount, syscall.MNT_DETACH|syscall.MNT_FORCE); err == nil {
			log.DEBUG.Println("umount:", mount)
		}
	}
	return
}

func LookupGroup(id string) (int, error) {
	var g *group.Group

	if _, err := strconv.Atoi(id); err == nil {
		g, err = group.LookupId(id)
		if err != nil {
			return -1, err
		}
	} else {
		g, err = group.Lookup(id)
		if err != nil {
			return -1, err
		}
	}

	return strconv.Atoi(g.Gid)
}

func SetGroup(id string) error {
	gid, err := LookupGroup(id)
	if err != nil {
		return err
	}
	return system.Setgid(gid)
}

func LookupUser(id string) (int, error) {
	var u *user.User

	if _, err := strconv.Atoi(id); err == nil {
		u, err = user.LookupId(id)
		if err != nil {
			return -1, err
		}
	} else {
		u, err = user.Lookup(id)
		if err != nil {
			return -1, err
		}
	}

	return strconv.Atoi(u.Uid)
}

func SetUser(id string) error {
	uid, err := LookupUser(id)
	if err != nil {
		return err
	}
	return system.Setuid(uid)
}

func DropCapabilities(keepCaps map[uint]bool) error {
	var i uint
	for i = 0; ; i++ {
		if keepCaps[i] {
			continue
		}
		if err := unix.Prctl(syscall.PR_CAPBSET_READ, uintptr(i), 0, 0, 0); err != nil {
			// Regard EINVAL as the condition of loop finish.
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.EINVAL {
				break
			}
			return err
		}
		if err := unix.Prctl(syscall.PR_CAPBSET_DROP, uintptr(i), 0, 0, 0); err != nil {
			// Ignore EINVAL since the capability may not be supported in this system.
			if errno, ok := err.(syscall.Errno); ok && errno == syscall.EINVAL {
				continue
			} else if errno, ok := err.(syscall.Errno); ok && errno == syscall.EPERM {
				return errors.New("required CAP_SETPCAP capabilities")
			} else {
				return err
			}
		}
	}

	if i == 0 {
		return errors.New("Failed to drop capabilities")
	}

	return nil
}
