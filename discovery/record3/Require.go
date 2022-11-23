package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const RequireRcmdStr = "require"

type Require struct {
	Name         string `@"require"`
	RequireRid   string `@STRING`
	requireCount uint32
}

func NewRequire(text string) (*Require, *errors.Error) {
	target, err := NewStruct(text, &Require{})
	if err != nil {
		return nil, err
	}
	return target.(*Require), nil
}

func (self *Require) ToString() string {
	return fmt.Sprintf("%s %s", self.Name, self.RequireRid)
}

func (self *Require) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Require) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	requirerid, err := context.ReplaceVariable(utils.Unquote(self.RequireRid))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	name, cate, err := utils.ParseRid(requirerid)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* 같은 record를 require 했는지 check
	 */
	tmpCurrentRid := utils.Rid(context.RecordName, context.RecordCategory)
	tmpRequireRid := utils.Rid(name, cate)

	if tmpCurrentRid == tmpRequireRid {
		fmt.Println("\n* Skip require, same record")
		return nil, nil
	}

	requireRecord, err := NewRecord(name, cate)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* replayer context 생성 및 실행
	 */
	logdir := fmt.Sprintf("%s/_require_/%d", context.LogDir, self.requireCount)
	for {
		if !utils.IsExist(logdir) {
			break
		}
		self.requireCount += 1
		logdir = fmt.Sprintf("%s/_require_/%d", context.LogDir, self.requireCount)
	}

	/* result 출력 옵션 처리
	 */
	printflag, depthindent := context.GetResultOptions()
	depthindent1 := depthindent
	depthindent += "  "

	rcdresult, err := NewRecordResult(0, tmpRequireRid, printflag, depthindent)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	requireContext, err := NewReplayerContext(name, cate, logdir, context.OutputPrintFlag,
		rcdresult, context.ForceEnvId, context.NoEnvHashCheck, context.Args)

	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	defer requireContext.Close()

	/* VarMapSlice 상속
	 */
	requireContext.VarMapSlice = append(requireContext.VarMapSlice, context.VarMapSlice...)
	requireContext.PushVarMapSlice(NewVariableMap())

	if printflag {
		fmt.Println()
		fmt.Printf("%s %s[r] require \"%s\"%s", depthindent1, constdef.ANSI_YELLOW, requirerid, constdef.ANSI_END)
	}

	defer func() {
		/* require 실행 결과를 context record result에 추가
		 */
		step, _ := NewStep(rcdresult)
		context.RecordResult.AddStep(step)
	}()
	self.requireCount += 1

	err = requireRecord.Play(requireContext)
	if err != nil {
		err = err.AddMsg(self.ToString())

		errorresult, err1 := NewErrorResult(err, requireContext, printflag, depthindent)
		if err1 != nil {
			return nil, err1.AddMsg(self.ToString())
		}
		rcdresult.SetResult(errorresult)
		return nil, err
	}

	err = requireRecord.Checker(requireContext)
	if err != nil {
		err = err.AddMsg(self.ToString())

		errorresult, err1 := NewErrorResult(err, requireContext, printflag, depthindent)
		if err1 != nil {
			return nil, err1.AddMsg(self.ToString())
		}
		rcdresult.SetResult(errorresult)
		return nil, err
	}

	rcdresult.SetResult(nil)
	return nil, nil
}

func (self *Require) GetName() string {
	return self.Name
}

func (self *Require) Dump() {
	repr.Println(self)
}
