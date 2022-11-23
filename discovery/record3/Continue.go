package record3

import (
	"discovery/errors"
	"github.com/alecthomas/repr"
)

const ContinueRcmdStr = "continue"

type Continue struct {
	Name string `@"continue"`
}

func (self *Continue) ToString() string {
	return self.Name
}

func (self *Continue) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Continue) Do(context *ReplayerContext) (Void, *errors.Error) {
	return CF_CONTINUE, nil
}

func (self *Continue) GetName() string {
	return self.Name
}

func (self *Continue) Dump() {
	repr.Println(self)
}
