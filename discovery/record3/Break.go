package record3

import (
	"discovery/errors"
	"github.com/alecthomas/repr"
)

// Control Flow 명령어 정의
const (
	CF_NOP int = iota
	CF_BREAK
	CF_CONTINUE
	CF_RETURN
)

const BreakRcmdStr = "break"

type Break struct {
	Name string `@"break"`
}

func (self *Break) ToString() string {
	return self.Name
}

func (self *Break) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Break) Do(context *ReplayerContext) (Void, *errors.Error) {
	return CF_BREAK, nil
}

func (self *Break) GetName() string {
	return self.Name
}

func (self *Break) Dump() {
	repr.Println(self)
}
