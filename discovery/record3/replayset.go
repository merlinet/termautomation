package record3

import (
	"bufio"
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type ReplaySet struct {
	Type       string
	Name       string
	RecordList []*Record

	LogDir         string   // 로그 디렉토리
	ForceEnvId     string   // override env id
	NoEnvHashCheck bool     // true인 경우 env hash를 하지 않음
	Args           []string // replayer arguments 값
}

func NewReplaySet(setname string, logdir string,
	forceEnvId string, noEnvHashCheck bool, args []string) (*ReplaySet, *errors.Error) {

	if len(setname) == 0 || len(logdir) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	set := ReplaySet{
		Type: "set",
		Name: setname,

		LogDir:         logdir,
		ForceEnvId:     forceEnvId,
		NoEnvHashCheck: noEnvHashCheck,
		Args:           args,
	}

	path, err := config.GetContentsReplaySetFilePath(set.Name)
	if err != nil {
		return nil, err
	}

	err = set.Load(path)
	if err != nil {
		return nil, err
	}

	return &set, nil
}

func NewReplayWithFailSet(resultTimeDir string, logdir string,
	forceEnvId string, noEnvHashCheck bool, args []string) (*ReplaySet, *errors.Error) {

	if len(resultTimeDir) == 0 || len(logdir) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	set := ReplaySet{
		Type: "failset",
		Name: fmt.Sprintf("%s - Incomplete record set", resultTimeDir),

		LogDir:         logdir,
		ForceEnvId:     forceEnvId,
		NoEnvHashCheck: noEnvHashCheck,
		Args:           args,
	}

	resultDir, err := config.GetContentsResultsDir()
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/%s/incomplete.set", resultDir, resultTimeDir)

	err = set.Load(path)
	if err != nil {
		return nil, err
	}

	return &set, nil
}

func NewReplaySetWithRids(rids []string, logdir string,
	forceEnvId string, noEnvHashCheck bool, args []string) (*ReplaySet, *errors.Error) {

	if len(rids) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	set := ReplaySet{
		Type: "rid",

		LogDir:         logdir,
		ForceEnvId:     forceEnvId,
		NoEnvHashCheck: noEnvHashCheck,
		Args:           args,
	}

	err := set.LoadWithRids(rids)
	if err != nil {
		return nil, err
	}

	set.Name = rids[0]

	return &set, nil
}

func NewReplaySetWithRecordFiles(recordfiles []string, logdir string,
	forceEnvId string, noEnvHashCheck bool, args []string) (*ReplaySet, *errors.Error) {

	if len(recordfiles) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	set := ReplaySet{
		Type: "file",

		LogDir:         logdir,
		ForceEnvId:     forceEnvId,
		NoEnvHashCheck: noEnvHashCheck,
		Args:           args,
	}

	err := set.LoadWithRecordFiles(recordfiles)
	if err != nil {
		return nil, err
	}

	set.Name = recordfiles[0]

	return &set, nil
}

func (self *ReplaySet) Load(path string) *errors.Error {
	if len(path) == 0 {
		return errors.New("invalid arguments")
	}

	fp, goerr := os.Open(path)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	reader := bufio.NewReader(fp)

	rids := []string{}
	for {
		data, _, goerr := reader.ReadLine()
		if goerr != nil {
			if goerr == io.EOF {
				break
			}
			return errors.New(fmt.Sprintf("%s", goerr))
		}

		rid := strings.TrimSpace(string(data))
		if len(rid) == 0 || rid[0] == ';' {
			continue
		}

		rids = append(rids, rid)
	}

	return self.LoadWithRids(rids)
}

func (self *ReplaySet) LoadWithRecordFiles(recordFiles []string) *errors.Error {
	if len(recordFiles) == 0 {
		return errors.New("Invalid arguments")
	}

	contentsDir, err := config.GetContentsDir()
	if err != nil {
		return err
	}

	rids := []string{}
	for _, filename := range recordFiles {
		if strings.HasSuffix(filename, ".record") == false {
			return errors.New(fmt.Sprintf(`Invalid record filename, "%s" doesn't have .record file extention`, filename))
		}

		recordpath, oserr := filepath.Abs(filename)
		if oserr != nil {
			return errors.New(fmt.Sprintf("%s", oserr))
		}

		if _, oserr := os.Stat(recordpath); oserr != nil {
			return errors.New(fmt.Sprintf("%s", oserr))
		}

		if strings.HasPrefix(recordpath, contentsDir) == false {
			return errors.New("Wrong contents working directory")
		}

		recordpath = strings.Replace(recordpath, contentsDir+"/", "", -1)
		rid := strings.Replace(recordpath, ".record", "", -1)
		rids = append(rids, rid)
	}

	return self.LoadWithRids(rids)
}

func (self *ReplaySet) LoadWithRids(rids []string) *errors.Error {
	if len(rids) == 0 {
		return errors.New("Invalid arguments")
	}

	for _, rid := range rids {
		name, cate, err := utils.ParseRid(rid)
		if err != nil {
			return err
		}

		record, err := NewRecord(name, cate)
		if err != nil {
			return err
		}

		self.RecordList = append(self.RecordList, record)
	}

	return nil
}

func (self *ReplaySet) Play(result *Result) *errors.Error {
	if result == nil {
		return errors.New("invalid arguments")
	}

	/* result 를 화면에 출력할지 여부, depth indent 초기 설정
	 */
	printflag := true
	depthindent := ""

	for seq, record := range self.RecordList {
		rid := utils.Rid(record.Name, record.Category)
		recordLogDir := fmt.Sprintf("%s/%d", self.LogDir, seq)

		/* record result 생성하고, result 에 추가
		 */
		rcdresult, err := NewRecordResult(uint32(seq), rid, printflag, depthindent)
		if err != nil {
			return err
		}
		result.AddRecordResult(rcdresult)

		context, err := NewReplayerContext(record.Name, record.Category,
			recordLogDir, false, rcdresult, self.ForceEnvId, self.NoEnvHashCheck, self.Args)

		if err != nil {
			errorresult, err1 := NewErrorResult(err, nil, printflag, depthindent)
			if err1 != nil {
				return err1
			}
			/* result 설정 및 summary 출력
			 */
			rcdresult.SetResult(errorresult)
			rcdresult.CountResult()
			rcdresult.PrintSummary()

			continue
		}

		err = record.Play(context)
		if err != nil {
			errorresult, err1 := NewErrorResult(err, context, printflag, depthindent)
			if err1 != nil {
				return err1
			}
			/* result 설정 및 summary 출력
			 */
			rcdresult.SetResult(errorresult)
			rcdresult.CountResult()
			rcdresult.PrintSummary()

			context.Close()
			continue
		}

		err = record.Checker(context)
		if err != nil {
			errorresult, err1 := NewErrorResult(err, context, printflag, depthindent)
			if err1 != nil {
				return err1
			}
			/* result 설정 및 summary 출력
			 */
			rcdresult.SetResult(errorresult)
			rcdresult.CountResult()
			rcdresult.PrintSummary()

			context.Close()
			continue
		}

		/* result 설정 및 summary 출력
		 */
		rcdresult.SetResult(nil)
		rcdresult.CountResult()
		rcdresult.PrintSummary()

		context.Close()
	}

	return nil
}

func (self *ReplaySet) Dump() {
	repr.Println(self)
}
