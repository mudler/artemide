package chroot

import (
	"os"
	"syscall"

	jww "github.com/spf13/jwalterweatherman"

	"github.com/docker/docker/pkg/mount"
	"github.com/mudler/artemide/pkg/osutil"
)

var keepCaps = map[uint]bool{
	0:  true, // CAP_CHOWN
	1:  true, // CAP_DAC_OVERRIDE
	2:  true, // CAP_DAC_READ_SEARCH
	3:  true, // CAP_FOWNER
	6:  true, // CAP_SETGID
	7:  true, // CAP_SETUID
	10: true, // CAP_NET_BIND_SERVICE
}

func Run(rootDir string, commands []string, files []string, bind []string, robind []string, nodropcaps bool) error {

	if !osutil.ExistsDir(rootDir) {
		return jww.FATAL.Errorf("No such directory %s:", rootDir)
	}

	var err error
	uid, gid := os.Getuid(), os.Getgid()

	// if group := c.String("group"); group != "" {
	// 	if gid, err = osutil.LookupGroup(group); err != nil {
	// 		return jww.FATAL.Errorf("Failed to lookup group:", err)
	// 	}
	// }
	// if user := c.String("user"); user != "" {
	// 	if uid, err = osutil.LookupUser(user); err != nil {
	// 		return jww.FATAL.Errorf("Failed to lookup user:", err)
	// 	}
	// }

	// copy files
	for _, f := range files {
		srcFile, destFile := fp.Join("/", f), fp.Join(rootDir, f)
		if err := osutil.Cp(srcFile, destFile); err != nil {
			return jww.FATAL.Errorf("Failed to copy %s:", f, err)
		}
		if err := os.Lchown(destFile, uid, gid); err != nil {
			return jww.FATAL.Errorf("Failed to lchown %s:", f, err)
		}
	}

	// mount -t proc none {{rootDir}}/proc
	if err := mount.Mount("none", fp.Join(rootDir, "/proc"), "proc", ""); err != nil {
		return jww.FATAL.Errorf("Failed to mount /proc: %s", err)
	}
	// mount --rbind /sys {{rootDir}}/sys
	if err := mount.Mount("/sys", fp.Join(rootDir, "/sys"), "none", "rbind"); err != nil {
		return jww.FATAL.Errorf("Failed to mount /sys: %s", err)
	}

	for _, dir := range bind {
		if err := bindMount(dir, rootDir, false); err != nil {
			return jww.FATAL.Errorf("Failed to bind mount %s:", dir, err)
		}
	}
	for _, dir := range robind {
		if err := bindMount(dir, rootDir, true); err != nil {
			return jww.FATAL.Errorf("Failed to robind mount %s:", dir, err)
		}
	}

	// create symlinks
	if err := osutil.Symlink("../run/lock", fp.Join(rootDir, "/var/lock")); err != nil {
		return jww.FATAL.Errorf("Failed to symlink lock file:", err)
	}

	if err := createDevices(rootDir, uid, gid); err != nil {
		return jww.FATAL.Errorf("Failed to create devices:", err)
	}

	jww.DEBUG.Println("chroot", rootDir, command)

	if err := syscall.Chroot(rootDir); err != nil {
		return jww.FATAL.Errorf("Failed to chroot:", err)
	}
	if err := syscall.Chdir("/"); err != nil {
		return jww.FATAL.Errorf("Failed to chdir /:", err)
	}

	if !nodropcaps {
		jww.DEBUG.Println("drop capabilities")
		if err := osutil.DropCapabilities(keepCaps); err != nil {
			return jww.FATAL.Errorf("Failed to drop capabilities:", err)
		}
	}

	jww.DEBUG.Println("setgid", gid)
	if err := osutil.Setgid(gid); err != nil {
		return jww.FATAL.Errorf("Failed to set group %d:", gid, err)
	}
	jww.DEBUG.Println("setuid", uid)
	if err := osutil.Setuid(uid); err != nil {
		return jww.FATAL.Errorf("Failed to set user %d:", uid, err)
	}

	return osutil.Execv(commands, os.Environ())
}

func Umount(rootDir string) {

	if !osutil.ExistsDir(rootDir) {
		return jww.FATAL.Errorf("No such directory %s", rootDir)
	}

	return osutil.UmountRoot(rootDir)
}
