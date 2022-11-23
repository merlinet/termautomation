package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const ErrorRcmdStr = "error"

type Error struct {
	Name    string `@"error"`
	Message string `@STRING`
}

func NewError(text string) (*Error, *errors.Error) {
	target, err := NewStruct(text, &Error{})
	if err != nil {
		return nil, err
	}
	return target.(*Error), nil
}

func (self *Error) ToString() string {
	return fmt.Sprintf("%s %s", self.Name, self.Message)
}

func (self *Error) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Error) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	msg, err := context.ReplaceVariable(utils.Unquote(self.Message))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	return nil, errors.New(msg)
}

func (self *Error) GetName() string {
	return self.Name
}

func (self *Error) Dump() {
	repr.Println(self)
}
