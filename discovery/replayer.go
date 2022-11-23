package main

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/fmt"
	"discovery/record3"
	"discovery/utils"
	"os"
)

func main() {
	recordVersion, err := utils.GetRecordVersionArg()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
		os.Exit(1)
	}

	/* record version 분기
	 */
	switch recordVersion {
	case "3":
		arg, err := record3.ParseReplayerArg()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
			os.Exit(1)
		}

		if arg.PrintWeb == false {
			config.Version()
		}

		replay, err := record3.NewReplayer(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
			os.Exit(1)
		}

		if arg.CheckGrammar {
			fmt.Println("OK: Grammar check succeeded")
			return
		}

		err = replay.Play()
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
			os.Exit(1)
		}
	default:
		fmt.Fprintln(os.Stderr, "ERR: invalid record version, you can specify record version 2 or 3")
		os.Exit(1)
	}
}
