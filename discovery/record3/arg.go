package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"flag"
	"os"
	"strings"
)

/* recorder arg
 */
type RecorderArg struct {
	NewFlag     bool
	CheckerFlag bool
	Rid         string
	EnvId       string
	InitRcmd    string
	InitRcmdArg string
}

func ParseRecorderArg() (*RecorderArg, *errors.Error) {
	newPtr := flag.String("new", "", "record id")
	ridPtr := flag.String("rid", "", "record id")
	envIdPtr := flag.String("env", "", "env id")
	connectPtr := flag.String("connect", "", "connect to host")
	spawnPtr := flag.String("spawn", "", "command to record")
	checkerPtr := flag.Bool("checker", false, "create checker shell script")

	flag.Parse()

	if len(*newPtr) > 0 {
		if len(*ridPtr) > 0 || len(*connectPtr) > 0 || len(*spawnPtr) > 0 {
			HelpRecorderArg()
			return nil, errors.New("Invalid arguments")
		}

		/* new flag 인 경우 envirinment 설정된 record 파일 생성
		 */
		if len(*envIdPtr) > 0 {
			envName, envCate, err := utils.ParseRid(*envIdPtr)
			if err != nil {
				return nil, err
			}

			envConfPath, err := config.GetContentsEnvFilePath(envCate, envName)
			if err != nil {
				return nil, err
			}

			if _, goerr := os.Stat(envConfPath); goerr != nil {
				return nil, errors.New(fmt.Sprintf("%s", goerr))
			}
		} else {
			return nil, errors.New("Invalid -env arguments")
		}

		arg := RecorderArg{
			NewFlag:     true,
			CheckerFlag: *checkerPtr,
			Rid:         strings.TrimSpace(*newPtr),
			EnvId:       strings.TrimSpace(*envIdPtr),
		}
		return &arg, nil

	} else if len(*ridPtr) > 0 {
		if len(*newPtr) > 0 || len(*envIdPtr) > 0 || *checkerPtr {
			HelpRecorderArg()
			return nil, errors.New("Invalid arguments")
		}

		arg := RecorderArg{
			NewFlag:     false,
			CheckerFlag: false,
			Rid:         strings.TrimSpace(*ridPtr),
		}

		if len(*connectPtr) > 0 && len(*spawnPtr) > 0 {
			HelpRecorderArg()
			return nil, errors.New("Invalid -connect, -spawn arguments")
		} else if len(*connectPtr) > 0 && len(*spawnPtr) == 0 {
			arg.InitRcmd = "connect"
			arg.InitRcmdArg = *connectPtr
		} else if len(*spawnPtr) > 0 && len(*connectPtr) == 0 {
			arg.InitRcmd = "spawn"
			arg.InitRcmdArg = *spawnPtr
		} else {
			HelpRecorderArg()
			return nil, errors.New("Invalid arguments")
		}

		return &arg, nil
	} else {
		HelpRecorderArg()
		return nil, errors.New("Invalid arguments")
	}
}

func HelpRecorderArg() {
	fmt.Println("Recorder")
	fmt.Println("  -new record id")
	fmt.Println("  -rid record id")
	fmt.Println("  -env env id")
	fmt.Println("  -envbase env base directory")
	fmt.Println("  -connect node name")
	fmt.Println("  -spawn command")
	fmt.Println("  -checker, creating checker script with when create new record")
	fmt.Println(`ex) recorder -new "network/route/test5" -env single_route`)
	fmt.Println(`    recorder -rid "network/route/test5" -connect UTM`)
	fmt.Println(`    recorder -rid "network/route/test5" -spawn "ssh root@192.168.60.66"`)
}

/* replayer arg
 */
type ReplayerArg struct {
	TestName       string
	SetName        string   // -set "set"
	FailSet        string   // -failset "timestamp"
	Rid            string   // -rid "rid" 형식
	RecordFile     string   // -f "example.record" 파일 지정
	ForceEnvId     string   // record 파일에 있는 environment overwrite
	NoEnvHashCheck bool     // record environment check sum 검사 안함
	Args           []string // replayer 초기 arguments
	CheckGrammar   bool     // rcmd syntax check 만 수행
	PrintWeb       bool
	LogDir         string
}

func ParseReplayerArg() (*ReplayerArg, *errors.Error) {
	testNamePtr := flag.String("name", "", "test name")
	setNamePtr := flag.String("set", "", "set name")
	failSetPtr := flag.String("failset", "", "result timestamp")
	ridPtr := flag.String("rid", "", "rid")
	recordFilePtr := flag.String("f", "", "record file")
	forceEnvIdPtr := flag.String("env", "", "force overwrite EnvId")
	noEnvHashCheckPtr := flag.Bool("noenvhashcheck", false, "doesn't check env hash value")
	argSepPtr := flag.String("argsep", " ", "arguments separator, default ' '")
	argsPtr := flag.String("args", "", "replayer arguments")
	checkGrammarPtr := flag.Bool("check", false, "check record grammar")
	printWebPtr := flag.Bool("web", false, "print output for web ui")
	logDirPtr := flag.String("logdir", "", "log directory")

	flag.Parse()

	if len(*testNamePtr) == 0 {
		HelpReplayerArg()
		return nil, errors.New("Invalid -name arguments")
	}

	if len(*setNamePtr) == 0 && len(*failSetPtr) == 0 && len(*ridPtr) == 0 && len(*recordFilePtr) == 0 {
		HelpReplayerArg()
		return nil, errors.New("Invalid -set, -failset, -rid or -f  arguments")
	}

	arg := ReplayerArg{
		TestName:       *testNamePtr,
		SetName:        strings.TrimSpace(*setNamePtr),
		FailSet:        strings.TrimSpace(*failSetPtr),
		Rid:            strings.TrimSpace(*ridPtr),
		RecordFile:     strings.TrimSpace(*recordFilePtr),
		ForceEnvId:     strings.TrimSpace(*forceEnvIdPtr),
		NoEnvHashCheck: *noEnvHashCheckPtr,
		CheckGrammar:   *checkGrammarPtr,
		PrintWeb:       *printWebPtr,
		LogDir:         *logDirPtr,
	}

	if len(*argsPtr) > 0 && len(*argSepPtr) > 0 {
		/* replayer 실행시 command 라인으로 arguments 받을수 있도록 함
		 * 내부 default 변수 ARGS 이름의 array변수 생성
		 */
		args := strings.Split(*argsPtr, *argSepPtr)
		for _, elem := range args {
			arg.Args = append(arg.Args, strings.TrimSpace(elem))
		}
	}

	return &arg, nil
}

func HelpReplayerArg() {
	fmt.Println("Replayer")
	fmt.Println("  -name test name")
	fmt.Println("  -set set name")
	fmt.Println("  -failset result timestamp")
	fmt.Println("  -rid rid name")
	fmt.Println("  -f record file")
	fmt.Println("  -env environment id, force overwrite record's envid ")
	fmt.Println("  -noenvhashcheck doesn't check env hash value, default false")
	fmt.Println("  -argsep argument string separator, default ' '")
	fmt.Println("  -args replayer arguments, replayer arguments")
	fmt.Println("  -check, check record grammar")
	fmt.Println("  -web output format for web ui")
	fmt.Println("  -logdir log directory, default current timestamp")
	fmt.Println(`ex) replayer -name "patch1" -set network`)
	fmt.Println(`ex) replayer -name "patch1" -rid "Test/test"`)
	fmt.Println(`ex) replayer -name "patch1" -f test.record`)
}
