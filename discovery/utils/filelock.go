package utils

import (
	"discovery/errors"
	"discovery/fmt"
	"os"
	"syscall"
	_ "unsafe"
)

var flock = syscall.Flock_t{
	Type:   syscall.F_WRLCK,
	Start:  0,
	Len:    1,
	Whence: 0,
}

func DumpFlockT(self *syscall.Flock_t) {
	fmt.Println("Type:", self.Type)
	fmt.Println("Whence:", self.Whence)
	fmt.Println("Pad_cgo_0:", self.Pad_cgo_0)
	fmt.Println("Start:", self.Start)
	fmt.Println("Len:", self.Len)
	fmt.Println("Pid:", self.Pid)
	fmt.Println("Pad_cgo_1:", self.Pad_cgo_1)
	fmt.Println("")
}

func IsFileLocked(fp *os.File) (bool, *errors.Error) {
	got := flock
	goerr := syscall.FcntlFlock(uintptr(fp.Fd()), syscall.F_GETLK, &got)
	if goerr != nil {
		return false, errors.New(fmt.Sprintf("%s", goerr))
	}

	if got.Type == syscall.F_UNLCK || got.Pid == int32(os.Getpid()) {
		return false, nil
	}
	return true, nil
}

func SetFileLock(fp *os.File) *errors.Error {
	goerr := syscall.FcntlFlock(uintptr(fp.Fd()), syscall.F_SETLKW, &flock)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	fmt.Println("locked...")
	return nil
}

func SetFileUnlock(fp *os.File) *errors.Error {
	unlock := flock
	unlock.Type = syscall.F_UNLCK
	goerr := syscall.FcntlFlock(uintptr(fp.Fd()), syscall.F_SETLK, &flock)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	fmt.Println("unlocked...")
	return nil
}
