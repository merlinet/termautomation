package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
)

const DebugRcmdStr = "debug"

/* debug rcmd
 * debug rcmd를 수행하면 현재 context dump 메시지 출력
 * option 에 따라 선별적 출력
 */
type Debug struct {
	Name string `@"debug"`
}

func NewDebug() (*Debug, *errors.Error) {
	debug := Debug{
		Name: DebugRcmdStr,
	}
	return &debug, nil
}

func (self *Debug) ToString() string {
	return self.Name
}

func (self *Debug) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Debug) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	debugMsg := context.DumpToString()
	fmt.Println("\n>>> DEBUG, CURRENT CONTEXT:")
	fmt.Println(debugMsg)

	return nil, nil
}

func (self *Debug) GetName() string {
	return self.Name
}

func (self *Debug) Dump() {
	repr.Println(self)
}
