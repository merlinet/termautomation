package utils

import (
	"discovery/errors"
	"discovery/fmt"
	"os"
	"path/filepath"
)

func MakeParentDir(path string, pathisdir bool) *errors.Error {
	if len(path) == 0 {
		return errors.New("invalid arguments")
	}

	dir := ""
	if pathisdir {
		dir = path
	} else {
		dir = filepath.Dir(path)
	}

	if _, oserr := os.Stat(dir); oserr != nil {
		if os.IsNotExist(oserr) {
			oserr1 := os.MkdirAll(dir, 0755)
			if oserr1 != nil {
				return errors.New(fmt.Sprintf("%s", oserr1))
			}
			return nil
		}
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return errors.New(fmt.Sprintf("%s log directory already exist.", dir))
}

func IsExist(path string) bool {
	if _, oserr := os.Stat(path); oserr == nil {
		return true
	}
	return false
}
