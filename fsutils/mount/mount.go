// Copyright 2015-2016 Platina Systems, Inc. All rights reserved.
// Use of this source code is governed by a BSD-style license described in the
// LICENSE file.

package mount

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/platinasystems/go/flags"
	"github.com/platinasystems/go/parms"
)

// hack around syscall incorrect definition
const MS_NOUSER uintptr = (1 << 31)
const procFilesystems = "/proc/filesystems"

type mount struct{}

type fstabEntry struct {
	fsSpec  string
	fsFile  string
	fsType  string
	mntOpts string
}

type Filesystems struct {
	name string
	list []string
}

func New() mount { return mount{} }

var translations = []struct {
	name string
	bits uintptr
	set  bool
}{
	{"-read-only", syscall.MS_RDONLY, true},
	{"-read-write", syscall.MS_RDONLY, false},
	{"-suid", syscall.MS_NOSUID, false},
	{"-no-suid", syscall.MS_NOSUID, true},
	{"-dev", syscall.MS_NODEV, false},
	{"-no-dev", syscall.MS_NODEV, true},
	{"-exec", syscall.MS_NOEXEC, false},
	{"-no-exec", syscall.MS_NOEXEC, true},
	{"-synchronous", syscall.MS_SYNCHRONOUS, true},
	{"-no-synchronous", syscall.MS_SYNCHRONOUS, true},
	{"-remount", syscall.MS_REMOUNT, true},
	{"-mand", syscall.MS_MANDLOCK, true},
	{"-no-mand", syscall.MS_MANDLOCK, false},
	{"-dirsync", syscall.MS_DIRSYNC, true},
	{"-no-dirsync", syscall.MS_DIRSYNC, false},
	{"-atime", syscall.MS_NOATIME, false},
	{"-no-atime", syscall.MS_NOATIME, true},
	{"-diratime", syscall.MS_NODIRATIME, false},
	{"-no-diratime", syscall.MS_NODIRATIME, true},
	{"-bind", syscall.MS_BIND, true},
	{"-move", syscall.MS_MOVE, true},
	{"-silent", syscall.MS_SILENT, true},
	{"-loud", syscall.MS_SILENT, false},
	{"-posixacl", syscall.MS_POSIXACL, true},
	{"-no-posixacl", syscall.MS_POSIXACL, false},
	{"-bindable", syscall.MS_UNBINDABLE, false},
	{"-unbindable", syscall.MS_UNBINDABLE, true},
	{"-private", syscall.MS_PRIVATE, true},
	{"-slave", syscall.MS_SLAVE, true},
	{"-shared", syscall.MS_SHARED, true},
	{"-relatime", syscall.MS_RELATIME, true},
	{"-no-relatime", syscall.MS_RELATIME, false},
	{"-iversion", syscall.MS_I_VERSION, true},
	{"-no-iversion", syscall.MS_I_VERSION, false},
	{"-strictatime", syscall.MS_STRICTATIME, true},
	{"-no-strictatime", syscall.MS_STRICTATIME, false},
}

var filesystems struct {
	all, auto Filesystems
}

func (mount) String() string { return "mount" }
func (mount) Usage() string  { return "mount [OPTION]... DEVICE [DIRECTORY]" }

func (mount mount) Main(args ...string) error {
	flag, args := flags.New(args,
		"--fake",
		"-v",
		"-a",
		"-defaults",
		"-r",
		"-read-write",
		"-suid",
		"-no-suid",
		"-dev",
		"-no-dev",
		"-exec",
		"-no-exec",
		"-synchronous",
		"-no-synchronous",
		"-remount",
		"-mand",
		"-no-mand",
		"-dirsync",
		"-no-dirsync",
		"-atime",
		"-no-atime",
		"-diratime",
		"-no-diratime",
		"-bind",
		"-move",
		"-silent",
		"-loud",
		"-posixacl",
		"-no-posixacl",
		"-bindable",
		"-unbindable",
		"-private",
		"-slave",
		"-shared",
		"-relatime",
		"-no-relatime",
		"-iversion",
		"-no-iversion",
		"-strictatime",
		"-no-strictatime")
	parm, args := parms.New(args, "-match", "-o", "-t")
	if len(parm["-t"]) == 0 {
		parm["-t"] = "auto"
	}

	filesystems.all.name = "all"
	filesystems.auto.name = "auto"
	var err error
	if flag["-a"] {
		err = mount.all(flag, parm)
	} else {
		switch len(args) {
		case 0:
			err = mount.show()
		case 1:
			err = mount.fstab(args[0], flag, parm)
		case 2:
			err = mount.one(parm["-t"], args[0], args[1], flag,
				parm)
		default:
			err = fmt.Errorf("%v: unexpected", args[2:])
		}
	}
	return err
}

func (mount mount) all(flag flags.Flag, parm parms.Parm) error {
	fstab, err := mount.loadFstab()
	if err != nil {
		return err
	}
	for _, x := range fstab {
		err = mount.one(x.fsType, x.fsSpec, x.fsFile, flag, parm)
		if err != nil {
			break
		}
	}
	return err
}

func (mount mount) fstab(name string, flag flags.Flag, parm parms.Parm) error {
	fstab, err := mount.loadFstab()
	if err != nil {
		return err
	}
	for _, x := range fstab {
		if name == x.fsSpec || name == x.fsFile {
			return mount.one(x.fsType, x.fsSpec, x.fsFile,
				flag, parm)
		}
	}
	return nil
}

func (mount) loadFstab() ([]fstabEntry, error) {
	f, err := os.Open("/etc/fstab")
	if err != nil {
		return nil, err
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	var fstab []fstabEntry
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Index(line, "#") < 0 {
			fields := strings.Fields(line)
			fstab = append(fstab, fstabEntry{
				fsSpec:  fields[0],
				fsFile:  fields[1],
				fsType:  fields[2],
				mntOpts: fields[3],
			})
		}
	}
	return fstab, scanner.Err()
}

func (mount) one(t, dev, dir string, flag flags.Flag, parm parms.Parm) error {
	var flags uintptr
	if flag["-defaults"] {
		//  rw, suid, dev, exec, auto, nouser, async
		flags &^= syscall.MS_RDONLY
		flags &^= syscall.MS_NOSUID
		flags &^= syscall.MS_NODEV
		flags &^= syscall.MS_NOEXEC
		if t == "" {
			t = "auto"
		}
		flags |= MS_NOUSER
		flags |= syscall.MS_ASYNC
	}
	for _, x := range translations {
		if flag[x.name] {
			if x.set {
				flags |= x.bits
			} else {
				flags &^= x.bits
			}
		}
	}
	if flag["--fake"] {
		fmt.Println("Would mount", dev, "type", t, "at", dir)
		return nil
	}

	tryTypes := []string{t}
	if t == "auto" {
		tryTypes = filesystems.auto.List()
	}

	var err error
	for _, t := range tryTypes {
		err = syscall.Mount(dev, dir, t, flags, parm["-o"])
		if err == nil {
			if flag["-v"] {
				fmt.Println("Mounted", dev, "at", dir)
			}
			break
		}
	}
	if err != nil {
		return fmt.Errorf("%s: %v", dev, err)
	}
	return nil
}

func (mount) show() error {
	f, err := os.Open("/proc/mounts")
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		fmt.Print(fields[0], " on ", fields[1], " type ", fields[2],
			"(", fields[3], ")\n")

	}
	return scanner.Err()
}

func (fs *Filesystems) List() []string {
	if len(fs.list) > 0 {
		return fs.list
	}
	f, err := os.Open(procFilesystems)
	if err != nil {
		return fs.list
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "nodev") {
			if fs.name == "auto" {
				continue
			}
			line = strings.TrimPrefix(line, "nodev")
		}
		line = strings.TrimSpace(line)
		fs.list = append(fs.list, line)
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scan:", procFilesystems, err)
	}
	return fs.list
}

func (mount) Apropos() map[string]string {
	return map[string]string{
		"en_US.UTF-8": "activated a filesystem",
	}
}

func (mount) Man() map[string]string {
	return map[string]string{
		"en_US.UTF-8": `NAME
	mount - activate a filesystem

SYNOPSIS
	mount [OPTION]... [DEVICE DIR]

DESCRIPTION
	Mount a filesystem on a target directory.

OPTIONS
	--fake
	-v		verbose
	-a		all [-match MATCH[,...]]
	-t FSTYPE[,...]
	-o FSOPT[,...]

	Where MATCH, FSTYPE and FSOPT are comma separated lists.

FSTYPE
	May be anything listed in /proc/filesystems; for example:
	sysfs, ramfs, proc, tmpfs, devtmpfs, debugfs, securityfs,
	sockfs, pipefs, devpts, hugetlbfs, pstore, mqueue, btrfs,
	ext2, ext3, ext4, nfs, nfs4, nfsd, aufs

FILESYSTEM INDEPENDENT FLAGS
	-defaults	-read-write -dev -exec -suid
	-r		read only
	-read-write
	-suid		Obey suid and sgid bits
	-no-suid	Ignore suid and sgid bits
	-dev		Allow use of special device files
	-no-dev		Disallow use of special device files
	-exec		Allow program execution
	-no-exec	Disallow program execution
	-synchronous	Writes are synced at once
	-no-synchronous	Writes aren't synced at once 
	-remount	Alter flags of mounted filesystem
	-mand		Allow mandatory locks
	-no-mand	Disallow mandatory locks
	-dirsync	Directory modifications are synchronous
	-no-dirsync	Directory modifications are asynchronous
	-atime		Update inode access times
	-no-atime	Don't update inode access-times
	-diratime	Update directory access-times
	-no-diratime	Don't update directory access times
	-bind		Bind a file or directory
	-move		Relocate an existing mount point
	-silent
	-loud
	-posixacl	Filesystem doesn't apply umask
	-no-posixacl	Filesystem applies umask
	-bindable	Make mount point able to be bind mounted
	-unbindable	Make mount point unable to be bind mounted
	-private	Change to private subtree
	-slave		Change to slave subtree
	-shared		Change to shared subtree
	-relatime	Update atime relative to mtime/ctime
	-no-relatime	Disable relatime
	-iversion	Update inode I-Version field
	-no-iversion	Don't update inode I-Version field
	-strictatime	Always perform atime updates
	-no-strictatime	May skip atime updates`,
	}
}
