package main

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/record3"
	"flag"
	"os"
)

type arg struct {
	EnvId         string
	RecordVersion string
}

func parseArg() (*arg, *errors.Error) {
	envIdPtr := flag.String("env", "", "environment id")
	recordVersionPtr := flag.String("rv", "3", "record version") // default record version 3

	flag.Parse()

	if len(*envIdPtr) == 0 {
		return nil, errors.New("Invalid -env arguments")
	}

	arg1 := arg{
		EnvId:         *envIdPtr,
		RecordVersion: *recordVersionPtr,
	}

	return &arg1, nil
}

func main() {
	arg, err := parseArg()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
		os.Exit(1)
	}

	config.Version()

	switch arg.RecordVersion {
	case "3":
		env, err := record3.NewEnvironment2(arg.EnvId)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
			os.Exit(1)
		}

		fmt.Printf("ENV RCMD(%s): %s\n\n", arg.RecordVersion, env.ToString())
	default:
		fmt.Fprintln(os.Stderr, "ERR: invalid record version, you can specify record version 2 or 3")
		os.Exit(1)
	}
}
