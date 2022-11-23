package record3

import (
	"discovery/errors"
	"github.com/alecthomas/repr"
)

const ReturnRcmdStr = "return"

type Return struct {
	Name string `@"return"`
}

func (self *Return) ToString() string {
	return self.Name
}

func (self *Return) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Return) Do(context *ReplayerContext) (Void, *errors.Error) {
	return CF_RETURN, nil
}

func (self *Return) GetName() string {
	return self.Name
}

func (self *Return) Dump() {
	repr.Println(self)
}
