package record3

import (
	"discovery/errors"
	"github.com/alecthomas/repr"
	"regexp"
)

const IfRcmdStr = "if"

type ElseIf struct {
	ElseifKeyword string      `@"elseif"`
	Expr          *Expression `@@`
	RcmdList      *RcmdList   `[ @@ ]`

	RcmdObjList []RcmdInterface
}

func (self *ElseIf) ToString() string {
	text := self.ElseifKeyword
	if self.Expr != nil {
		text += " " + self.Expr.ToString()
	}

	/* XXX
	if self.RcmdList != nil {
		text += "\n"
		text += ToStringRcmdList(self.RcmdList)
	}
	*/

	return text
}

func (self *ElseIf) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *ElseIf) Do(context *ReplayerContext) (bool, Void, *errors.Error) {
	condition, err := CheckCondition(self.Expr, context)
	if err != nil {
		return false, nil, err.AddMsg(self.ToString())
	}

	if condition == true {
		controlflow, err := playIfRcmdList(self.RcmdObjList, context)
		if err != nil {
			return false, nil, err.AddMsg(self.ToString())
		}
		return true, controlflow, nil
	} else {
		return false, nil, nil
	}
}

type Else struct {
	ElseKeyword  string    `@"else"`
	RcmdList     *RcmdList `[ @@ ]`
	EndifKeyword string    `@"endif"`

	RcmdObjList []RcmdInterface
}

func (self *Else) ToString() string {
	text := self.ElseKeyword
	/* XXX
	if self.RcmdList != nil {
		text += "\n"
		text += ToStringRcmdList(self.RcmdList)
	}
	*/
	return text
}

func (self *Else) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *Else) Do(context *ReplayerContext) (Void, *errors.Error) {
	controlflow, err := playIfRcmdList(self.RcmdObjList, context)
	if err != nil {
		return controlflow, err.AddMsg(self.ToString())
	}
	return controlflow, nil
}

type If struct {
	Name         string      `@"if"`
	Expr         *Expression `@@`
	RcmdList     *RcmdList   `[ @@ ]`
	ElseIf       []*ElseIf   `{ @@ { @@ } }`
	Else         *Else       `( @@ `
	EndifKeyword string      ` |@"endif" )`

	RcmdObjList []RcmdInterface
}

func (self *If) ToString() string {
	text := self.Name
	if self.Expr != nil {
		text += " " + self.Expr.ToString()
	}

	/* XXX
	if self.RcmdList != nil {
		text += "\n"
		text += ToStringRcmdList(self.RcmdList)
	}

	for _, elseif := range self.ElseIf {
		text += "\n"
		text += elseif.ToString()
	}

	if self.Else != nil {
		text += "\n"
		text += self.Else.ToString()
	}

	text += self.EndifKeyword
	*/

	return text
}

func (self *If) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *If) Do(context *ReplayerContext) (Void, *errors.Error) {
	condition, err := CheckCondition(self.Expr, context)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	if condition == true {
		controlflow, err := playIfRcmdList(self.RcmdObjList, context)
		if err != nil {
			return controlflow, err.AddMsg(self.ToString())
		}
		return controlflow, nil
	}

	if len(self.ElseIf) > 0 {
		for _, elseif := range self.ElseIf {
			err := elseif.Prepare(context)
			if err != nil {
				return nil, err
			}

			end, controlflow, err := elseif.Do(context)
			if err != nil {
				return controlflow, err
			}

			if end {
				return controlflow, nil
			}
		}
	}

	if self.Else != nil {
		err := self.Else.Prepare(context)
		if err != nil {
			return nil, err
		}

		return self.Else.Do(context)
	}

	return nil, nil
}

func (self *If) GetName() string {
	return self.Name
}

func (self *If) Dump() {
	repr.Println(self)
}

func CheckCondition(expr *Expression, context *ReplayerContext) (bool, *errors.Error) {
	if expr == nil {
		return false, errors.New("invalid if condition expression")
	}

	res, err := expr.Do(context)
	if err != nil {
		return false, err
	}

	switch res.(type) {
	case bool:
		if res.(bool) == false {
			return false, nil
		}
	case string:
		if len(res.(string)) <= 0 {
			return false, nil
		}
	case *regexp.Regexp:
		if res.(*regexp.Regexp) == nil {
			return false, nil
		}
	case float64:
		if res.(float64) <= 0 {
			return false, nil
		}
	case []Void:
		if len(res.([]Void)) <= 0 {
			return false, nil
		}
	default:
		return false, errors.New("invalid if condition value")
	}

	return true, nil
}

func playIfRcmdList(rcmdobjlist []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	varmap := NewVariableMap()
	context.PushVarMapSlice(varmap)
	defer context.PopVarMapSlice()

	return PlayRcmdList(rcmdobjlist, context)
}
