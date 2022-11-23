package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
	"strings"
)

/* Set 정의
 */
const SetRcmdStr = "set"

type Set struct {
	Name    string      `@"set"`
	VarName string      `@IDENT`
	Expr    *Expression `@@`
}

func NewSet(text string) (*Set, *errors.Error) {
	target, err := NewStruct(text, &Set{})
	if err != nil {
		return nil, err
	}
	return target.(*Set), nil
}

func (self *Set) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.VarName, self.Expr.ToString())
}

func (self *Set) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Set) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Expr == nil {
		return nil, errors.New("invalid argument").AddMsg(self.ToString())
	}

	value, err := self.Expr.Do(context)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* set varname이 function name이랑 같은기 검사
	 */
	_, ok := context.FunctionMap[self.VarName]
	if ok {
		return nil, errors.New(fmt.Sprintf("can't set to '%s' function name as variable name", self.VarName)).AddMsg(self.ToString())
	}

	variable, err := context.GetVariable(self.VarName)
	if err == nil {
		err1 := variable.SetValue(value)
		if err1 != nil {
			return nil, err1.AddMsg(self.ToString())
		}
	} else {
		err := context.SetVariable(self.VarName, value, "")
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	}
	return nil, nil
}

func (self *Set) GetName() string {
	return self.Name
}

func (self *Set) Dump() {
	repr.Println(self)
}

/* Unset 정의
 */
const UnsetRcmdStr = "unset"

type Unset struct {
	Name          string       `@"unset"`
	PrimValueList []*PrimValue `{ @@ }`
}

func NewUnset(text string) (*Unset, *errors.Error) {
	target, err := NewStruct(text, &Unset{})
	return target.(*Unset), err
}

func (self *Unset) ToString() string {
	text := self.Name + " "
	for _, pv := range self.PrimValueList {
		text += pv.ToString() + " "
	}
	return strings.TrimSpace(text)
}

func (self *Unset) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Unset) Do(context *ReplayerContext) (Void, *errors.Error) {
	if len(self.PrimValueList) == 0 {
		return nil, errors.New("invalid unset syntax").AddMsg(self.ToString())
	}

	for _, pv := range self.PrimValueList {
		if pv.Value == nil || pv.Value.Variable == nil || len(*pv.Value.Variable) == 0 ||
			(pv.Param != nil && pv.Param.FuncParam != nil) {

			return nil, errors.New("invalid unset syntax").AddMsg(self.ToString())
		}
		varName := *pv.Value.Variable

		/* check function name
		 */
		_, ok := context.FunctionMap[varName]
		if ok {
			return nil, errors.New(fmt.Sprintf("'%s' is function name", varName)).AddMsg(self.ToString())
		}

		variable, err := context.GetVariable(varName)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		/* primary parameter가 없으면 변수 삭제
		 */
		if pv.Param == nil || pv.Param.Idx == nil {
			err := context.DelVariable(varName)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
		} else {
			err = pv.Param.DeleteParamValue(&variable.Value, context)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
		}
	}

	return nil, nil
}

func (self *Unset) GetName() string {
	return self.Name
}

func (self *Unset) Dump() {
	repr.Println(self)
}

/* 기존 set 대체 set rcmd
 * array, map 에 값 재할당 가능하도록 함
 * ex) seta arr[0] = "address"
 * ex) seta map["key"] = "address"
 */
const SetaRcmdStr = "seta"

type Seta struct {
	Name          string      `@"seta"`
	LeftPrimValue *PrimValue  `@@ "="`
	RightExpr     *Expression `@@`
}

func NewSeta(text string) (*Seta, *errors.Error) {
	target, err := NewStruct(text, &Seta{})
	if err != nil {
		return nil, err
	}
	return target.(*Seta), nil
}

func (self *Seta) ToString() string {
	return fmt.Sprintf("%s %s = %s", self.Name, self.LeftPrimValue.ToString(), self.RightExpr.ToString())
}

func (self *Seta) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Seta) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.RightExpr == nil {
		return nil, errors.New("invalid seta syntax").AddMsg(self.ToString())
	}

	rightValue, err := self.RightExpr.Do(context)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	lpv := self.LeftPrimValue

	/* 좌변은 primary의 variable 타임이어야 함
	 */
	if lpv.Value == nil || lpv.Value.Variable == nil || len(*lpv.Value.Variable) == 0 {
		return nil, errors.New("invalid seta syntax").AddMsg(self.ToString())
	}
	varName := *lpv.Value.Variable

	/* set varname이 function name이랑 같은기 검사
	 */
	_, ok := context.FunctionMap[varName]
	if ok {
		return nil, errors.New(fmt.Sprintf("can't set to '%s' function name as variable name", varName)).AddMsg(self.ToString())
	}

	variable, err := context.GetVariable(varName)
	if err == nil {
		if lpv.Param == nil {
			variable.Value = rightValue
		} else if lpv.Param.FuncParam != nil {
			return nil, errors.New("invalid seta syntax").AddMsg(self.ToString())
		} else {
			/* variable Value 값 갱신
			 */
			err := lpv.Param.SetParamValue(&variable.Value, rightValue, context)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
		}
	} else {
		/* new variable 생성
		 */
		if lpv.Param != nil {
			return nil, errors.New(fmt.Sprintf("'%s' is invalid variable name, parameter can place after valid variable name", varName)).AddMsg(self.ToString())
		}

		err := context.SetVariable(varName, rightValue, "")
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	}

	return nil, nil
}

func (self *Seta) GetName() string {
	return self.Name
}

func (self *Seta) Dump() {
	repr.Println(self)
}
