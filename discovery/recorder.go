package main

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/fmt"
	"discovery/record3"
	"os"
)

func main() {
	config.Version()

	arg, err := record3.ParseRecorderArg()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
		os.Exit(1)
	}

	if arg.NewFlag {
		/* Environment 추가된 record 파일 생성
		 */
		err := record3.CreateNewRecord(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
			os.Exit(1)
		}
		fmt.Println("Done.")
	} else {
		/* record 파일의 Environment를 참조하여 recording 시작
		 */
		err := record3.StartRecorder(arg)
		if err != nil {
			fmt.Fprintln(os.Stderr, "ERR:", err.ToString(constdef.DEBUG), ", Abort")
			os.Exit(1)
		}
	}
}
