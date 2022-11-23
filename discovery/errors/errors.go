package errors

import (
	"discovery/constdef"
	"discovery/fmt"
	"path"
	"runtime"
	"strings"
)

type Error struct {
	File      string
	Func      string
	Line      int
	Msg       string
	CheckCode uint8 `json:"-"` /* record2 호환 */
}

const (
	depthOfFunctionCaller = 1
)

func New(msg string) *Error {
	pc, file, line, _ := runtime.Caller(depthOfFunctionCaller)
	fn := runtime.FuncForPC(pc)
	elems := strings.Split(fn.Name(), ".")

	funcstr := ""
	if len(elems) > 0 {
		funcstr = elems[len(elems)-1]
	}

	err := Error{
		File:      path.Base(file),
		Func:      funcstr,
		Line:      line,
		Msg:       msg,
		CheckCode: constdef.CHECKER_NA,
	}

	return &err
}

func (self *Error) ToString(debugFlag bool) string {
	if debugFlag {
		return fmt.Sprintf("(%s:%d) %s(), %s", self.File, self.Line, self.Func, self.Msg)
	} else {
		return fmt.Sprintf("%s", self.Msg)
	}
}

func (self *Error) AddMsg(msg string) *Error {
	if len(msg) > 0 {
		self.Msg = fmt.Sprintf("%s -> %s", msg, self.Msg)
	}
	return self
}

func (self *Error) AppendMsg(msg string) *Error {
	if len(msg) > 0 {
		self.Msg = fmt.Sprintf("%s -> %s", self.Msg, msg)
	}
	return self
}
