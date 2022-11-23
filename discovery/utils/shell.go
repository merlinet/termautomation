package utils

import (
	"discovery/errors"
	"discovery/fmt"
	"os"
	"os/exec"
	"strings"
)

func ExecShell(cwd string, command string) ([]string, *errors.Error) {
	cmd := exec.Command("/bin/sh", "-c", command)

	cmd.Env = append(os.Environ(), fmt.Sprintf("PATH=%s:%s", cwd, os.Getenv("PATH")))
	cmd.Dir = cwd

	out, oserr := cmd.CombinedOutput()
	if oserr != nil {
		return strings.Split(strings.TrimSpace(string(out)), "\n"), errors.New(fmt.Sprintf("%s", oserr))
	}

	return strings.Split(strings.TrimSpace(string(out)), "\n"), nil
}
