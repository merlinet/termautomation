package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
)

const CheckRcmdStr = "check"

type Check struct {
	Name     string      `@"check"`
	Expr     *Expression `@@`
	StepFlag bool        `[ [@"step"] ]` // check success/fail result 는 앞 comment에 붙여 출력 하지 않도록 함
}

func NewCheck(text string) (*Check, *errors.Error) {
	target, err := NewStruct(text, &Check{})
	if err != nil {
		return nil, err
	}
	return target.(*Check), nil
}

func (self *Check) ToString() string {
	return fmt.Sprintf("%s %s", self.Name, self.Expr.ToString())
}

func (self *Check) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Check) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("invalid arguments")
	}

	if self.Expr == nil {
		return nil, errors.New("invalid check expression")
	}

	chkResult := context.RecordResult.GetLastCheckStep()
	if chkResult == nil || self.StepFlag {
		printflag, depthindent := context.GetResultOptions()

		/* context record result 에 마지막 check result step 이 없을 경우 생성
		 */
		chkres, err := NewCheckResult("*", self.ToString(), printflag, depthindent)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		chkResult = chkres

		step, err := NewStep(chkResult)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		context.RecordResult.AddStep(step)
	}

	value, err := self.Expr.Do(context)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	var result bool

	switch value.(type) {
	case bool:
		result = value.(bool)
	default:
		return nil, errors.New("invalid check result type(not a bool)").AddMsg(self.ToString())
	}

	/* check condition string replace 실행 값을 step값, checkcondition으로 입력
	 */
	msg, err1 := context.ReplaceVariable(self.ToString())
	if err1 != nil {
		return nil, err1.AddMsg(self.ToString())
	}

	/* FAIL
	 */
	if result == false {
		chkResult.SetResult(constdef.FAIL)
		/* recorder에서 호출할 경우 false 리턴
		 */
		if context.RecorderFlag {
			return result, nil
		}

		chkResult.SetInfo(context.LastSendSessionName, context.LastSend,
			context.LastOutputSessionName, context.LastOutput, context.ExitCode, msg)

		if context.FailedButContinue {
			return nil, nil
		} else {
			return CF_RETURN, nil
		}
	}

	/* SUCCESS
	 */
	chkResult.SetResult(constdef.SUCCESS)

	/* recorder에서 호출할 경우 false 리턴
	 */
	if context.RecorderFlag {
		return result, nil
	}

	chkResult.SetInfo(context.LastSendSessionName, context.LastSend,
		context.LastOutputSessionName, context.LastOutput, context.ExitCode, msg)

	return nil, nil
}

func (self *Check) GetName() string {
	return self.Name
}

func (self *Check) Dump() {
	repr.Println(self)
}
