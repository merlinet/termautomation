package main

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/record3"
	"discovery/utils"
	"flag"
	"os"
)

type arg struct {
	Rid           string
	RecordVersion string
	PrintFileFlag bool
}

func parseArg() (*arg, *errors.Error) {
	ridPtr := flag.String("rid", "", "record id")
	recordVersionPtr := flag.String("rv", "3", "record id") // default record version 3
	printFileFlagPtr := flag.Bool("w", false, "enable write output to file")

	flag.Parse()

	if len(*ridPtr) == 0 {
		return nil, errors.New("Invalid -rid arguments")
	}

	arg1 := arg{
		Rid:           *ridPtr,
		RecordVersion: *recordVersionPtr,
		PrintFileFlag: *printFileFlagPtr,
	}

	return &arg1, nil
}

func main() {
	arg, err := parseArg()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
		os.Exit(1)
	}

	name, cate, err := utils.ParseRid(arg.Rid)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
		os.Exit(1)
	}

	/* record version 분기
	 */
	switch arg.RecordVersion {
	case "3":
		record, err := record3.NewRecord(name, cate)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
			os.Exit(1)
		}

		if arg.PrintFileFlag {
			pathPrefix, err := config.GetContentsRecordPrefix(name, cate)
			if err != nil {
				fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
				os.Exit(1)
			}

			specPath := fmt.Sprintf("%s.spec", pathPrefix)
			utils.MakeParentDir(specPath, false)
			oserr := fmt.InitPrint([]string{specPath})
			if oserr != nil {
				fmt.Fprintln(os.Stderr, "ERR:", oserr)
				os.Exit(1)
			}
		}

		err = record.Spec()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG))
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "ERR: invalid record version, you can specify record version 2 or 3")
		os.Exit(1)
	}
}
