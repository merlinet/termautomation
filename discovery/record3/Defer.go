package record3

import (
	"discovery/errors"
	"github.com/alecthomas/repr"
)

const DeferRcmdStr = "defer"

type Defer struct {
	Name            string    `@"defer"`
	RcmdList        *RcmdList `@@`
	EnddeferKeyword string    `@"enddefer"`

	RcmdObjList []RcmdInterface
}

func NewDefer(text string) (*Defer, *errors.Error) {
	target, err := NewStruct(text, &Defer{})
	if err != nil {
		return nil, err
	}
	return target.(*Defer), nil
}

func (self *Defer) ToString() string {
	text := self.Name + " ... "
	/* XXX
	if self.RcmdList != nil {
		text += ToStringRcmdList(self.RcmdList)
	}
	*/
	text += self.EnddeferKeyword
	return text
}

func (self *Defer) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *Defer) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	/* context defer list 에 추가
	 */
	context.DeferList = append(context.DeferList, self)
	return nil, nil
}

func (self *Defer) Do2(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguemnt")
	}

	controlflow, err := PlayRcmdList(self.RcmdObjList, context)
	if err != nil {
		return controlflow, err.AddMsg(self.ToString())
	}

	switch controlflow.(type) {
	case int:
		switch controlflow.(int) {
		case CF_BREAK, CF_CONTINUE:
			return controlflow, errors.New("break, continue can place after for, table rcmd").AddMsg(self.ToString())
		case CF_RETURN:
			return controlflow, nil
		}
	}
	return nil, nil
}

func (self *Defer) GetName() string {
	return self.Name
}

func (self *Defer) Dump() {
	repr.Println(self)
}
