package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const LogRcmdStr = "log"

type Log struct {
	Name        string `@"log"`
	OnFlag      *bool  `(@"on" | "off")`
	LogPrefix   string `@STRING`
	SessionName string `@IDENT`
}

func NewLog(text string) (*Log, *errors.Error) {
	target, err := NewStruct(text, &Log{})
	if err != nil {
		return nil, err
	}
	return target.(*Log), nil
}

func (self *Log) ToString() string {
	onoff := "off"
	if self.OnFlag != nil && *self.OnFlag {
		onoff = "on"
	}
	return fmt.Sprintf("%s %s %s %s", self.Name, onoff, self.LogPrefix, self.SessionName)
}

func (self *Log) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Log) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	sessionnode, err := context.GetSessionNode(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	sessionnode.LogFlag = false
	if self.OnFlag != nil && *self.OnFlag {
		sessionnode.LogFlag = true
	}
	sessionnode.LogPrefix, err = context.ReplaceVariable(utils.Unquote(self.LogPrefix))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	return nil, nil
}

func (self *Log) GetName() string {
	return self.Name
}

func (self *Log) Dump() {
	repr.Println(self)
}
