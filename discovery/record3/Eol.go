package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
	"strings"
)

const EolRcmdStr = "eol"

type Eol struct {
	Name        string `@"eol"`
	Eol         string `@("lf" | "cr" | "crlf")`
	SessionName string `@IDENT`
}

func NewEol(text string) (*Eol, *errors.Error) {
	target, err := NewStruct(text, &Eol{})
	if err != nil {
		return nil, err
	}
	return target.(*Eol), nil
}

func (self *Eol) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.Eol, self.SessionName)
}

func (self *Eol) Prepare(context *ReplayerContext) *errors.Error {
	switch strings.ToLower(self.Eol) {
	case constdef.EOL_CR:
	case constdef.EOL_LF:
	case constdef.EOL_CRLF:
	default:
		return errors.New("invalid eol argument")
	}
	return nil
}

func (self *Eol) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	sessionnode, err := context.GetSessionNode(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	switch strings.ToLower(self.Eol) {
	case constdef.EOL_CR:
		sessionnode.Proc.Eol = "\r"
	case constdef.EOL_LF:
		sessionnode.Proc.Eol = "\n"
	case constdef.EOL_CRLF:
		sessionnode.Proc.Eol = "\r\n"
	default:
		return nil, errors.New("invalid eol argument").AddMsg(self.ToString())
	}

	return nil, nil
}

func (self *Eol) GetName() string {
	return self.Name
}

func (self *Eol) Dump() {
	repr.Println(self)
}
