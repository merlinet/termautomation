package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
)

const CloseRcmdStr = "close"

type Close struct {
	Name        string `@"close"`
	SessionName string `@IDENT`
}

func NewClose(text string) (*Close, *errors.Error) {
	target, err := NewStruct(text, &Close{})
	if err != nil {
		return nil, err
	}
	return target.(*Close), nil
}

func (self *Close) ToString() string {
	return fmt.Sprintf("%s %s", self.Name, self.SessionName)
}

func (self *Close) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Close) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	session, ok := context.SessionMap[self.SessionName]
	if !ok {
		return nil, errors.New(fmt.Sprintf("%s, invalid session name", self.SessionName)).AddMsg(self.ToString())
	}

	if session != nil {
		session.Close()
	}
	delete(context.SessionMap, self.SessionName)
	return nil, nil
}

func (self *Close) GetName() string {
	return self.Name
}

func (self *Close) Dump() {
	repr.Println(self)
}
