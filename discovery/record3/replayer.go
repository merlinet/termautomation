package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"os"
	"time"
)

type Replayer struct {
	TestName  string
	SetName   string
	SetType   string
	TimeStamp string
	LogDir    string
	Set       *ReplaySet
}

/* replay set을 생성한다.
 * replay set은
 * 1. setname 있으면 set load
 * 2. rid로 생성
 * 3. recordfile로 생성
 */
func NewReplayer(arg *ReplayerArg) (*Replayer, *errors.Error) {
	if arg == nil || len(arg.TestName) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	dir, err := config.GetContentsResultsDir()
	if err != nil {
		return nil, err
	}

	t := time.Now()
	timestamp := t.Format("20060102150405")

	logdir := ""
	if len(arg.LogDir) > 0 {
		logdir = fmt.Sprintf("%s/%s", dir, arg.LogDir)
	} else {
		logdir = fmt.Sprintf("%s/%s", dir, timestamp)
	}

	screenLogPath := fmt.Sprintf("%s/screen.log", logdir)
	screenLogPathTmp := fmt.Sprintf("%s/%d.log", constdef.TMP_DIR, os.Getpid())

	/* grammar check 인 경우 디렉토리 생성 안함
	 */
	if arg.CheckGrammar == false {
		err = utils.MakeParentDir(screenLogPath, false)
		if err != nil {
			return nil, err
		}
		utils.MakeParentDir(screenLogPathTmp, false)

		var oserr error = nil
		/* stdout 출력 메시지를 WebUI에서 읽기 위한 처리
		 */
		if arg.PrintWeb {
			oserr = fmt.InitPrintWeb([]string{screenLogPath, screenLogPathTmp})
		} else {
			oserr = fmt.InitPrint([]string{screenLogPath, screenLogPathTmp})
		}
		if oserr != nil {
			return nil, errors.New(fmt.Sprintf("%s", oserr))
		}
	}

	set := &ReplaySet{}

	if len(arg.SetName) > 0 {
		set, err = NewReplaySet(arg.SetName, logdir, arg.ForceEnvId, arg.NoEnvHashCheck, arg.Args)
	} else if len(arg.FailSet) > 0 {
		set, err = NewReplayWithFailSet(arg.FailSet, logdir, arg.ForceEnvId, arg.NoEnvHashCheck, arg.Args)
	} else if len(arg.Rid) > 0 {
		set, err = NewReplaySetWithRids([]string{arg.Rid}, logdir, arg.ForceEnvId, arg.NoEnvHashCheck, arg.Args)
	} else if len(arg.RecordFile) > 0 {
		set, err = NewReplaySetWithRecordFiles([]string{arg.RecordFile}, logdir, arg.ForceEnvId, arg.NoEnvHashCheck, arg.Args)
	} else {
		err = errors.New("Invalid arguments")
	}

	if err != nil {
		return nil, err
	}

	replayer := Replayer{
		TestName:  arg.TestName,
		SetName:   set.Name,
		SetType:   set.Type,
		TimeStamp: timestamp,
		LogDir:    logdir,
		Set:       set,
	}

	return &replayer, nil
}

func (self *Replayer) Play() *errors.Error {
	if self.Set == nil {
		return errors.New("Invalid arguments")
	}
	set := self.Set

	result, err := NewResult(self.TestName, self.SetName, self.SetType, self.TimeStamp, self.LogDir)
	if err != nil {
		return err
	}

	result.PrintTitle()

	err = set.Play(result)
	if err != nil {
		return err
	}

	result.CountResult()
	result.PrintSummary()

	err = result.WriteJson()
	if err != nil {
		return err
	}

	err = result.WriteIncompleteRecordSet()
	if err != nil {
		return err
	}

	return nil
}
