package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const ScriptRcmdStr = "script"

/* script Rcmd는 local 명령어를 실행하게 한다
 * 옵션으로 checker 설정할 경우 checker로 동작, fail 발생 시 중단 된다
 */
type Script struct {
	Name        string  `@"script"`
	ScriptName  string  `@STRING`
	VarName     *string `[ [@IDENT]`        // script output string이 저장됨
	CheckerFlag bool    `  [@"checker"] ] ` // script가 checker 로 동작 할지 구분
}

func NewScript(text string) (*Script, *errors.Error) {
	target, err := NewStruct(text, &Script{})
	if err != nil {
		return nil, err
	}
	return target.(*Script), nil
}

func (self *Script) ToString() string {
	if self.CheckerFlag {
		if self.VarName != nil {
			return fmt.Sprintf("%s %s %s checker", self.Name, self.ScriptName, *self.VarName)
		} else {
			return fmt.Sprintf("%s %s checker", self.Name, self.ScriptName)
		}
	}

	if self.VarName != nil {
		return fmt.Sprintf("%s %s %s", self.Name, self.ScriptName, *self.VarName)
	} else {
		return fmt.Sprintf("%s %s", self.Name, self.ScriptName)
	}
}

func (self *Script) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Script) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	/* 문자열에 포함된 변수 replace
	 */
	scriptname, err := context.ReplaceVariable(utils.Unquote(self.ScriptName))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	cwd, err := config.GetContentsRecordDir(context.RecordCategory)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	printflag, depthindent := context.GetResultOptions()

	if self.CheckerFlag {
		/* script checker 옵션에서 checker result 생성
		 */
		checkerResult, err := NewCheckerResult(scriptname, printflag, depthindent)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		step, err := NewStep(checkerResult)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		context.RecordResult.AddStep(step)

		/* shell 명령어 실행
		 */
		output, err := utils.ExecShell(cwd, scriptname)
		if err != nil {
			checkerResult.SetResult(output, constdef.FAIL)

			if context.FailedButContinue {
				return nil, nil
			} else {
				return CF_RETURN, nil
			}
		}

		checkerResult.SetResult(output, constdef.SUCCESS)

		if self.VarName != nil {
			err := context.SetVariable(*self.VarName, output, "")
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
		}

		return nil, nil // 처리
	}

	/* checker 옵션 없을 경우
	 */
	output, _ := utils.ExecShell(cwd, scriptname)

	if self.VarName != nil {
		err := context.SetVariable(*self.VarName, output, "")
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	}

	return nil, nil
}

func (self *Script) GetName() string {
	return self.Name
}

func (self *Script) Dump() {
	repr.Println(self)
}
