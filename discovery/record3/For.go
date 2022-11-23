package record3

import (
	"discovery/errors"
	"fmt"
	"github.com/alecthomas/repr"
	"reflect"
)

const ForRcmdStr = "for"

type ForInCondition struct {
	FirstVarName  string      `@IDENT`
	SecondVarName string      `"," @IDENT`
	InKeyword     string      `@"in"`
	Value         *Expression `@@`
}

func (self *ForInCondition) ToString() string {
	text := self.FirstVarName + ", " + self.SecondVarName + " " + self.InKeyword + " "
	if self.Value != nil {
		text += self.Value.ToString()
	}
	return text
}

func (self *ForInCondition) Do(rcmdobjlist []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	varmap := NewVariableMap()
	context.PushVarMapSlice(varmap)
	defer context.PopVarMapSlice()

	if self.Value == nil {
		return nil, errors.New("invalid 'for in' condition")
	}

	value, err := self.Value.Do(context)
	if err != nil {
		return nil, err
	}

	controlflow, err := doInCondition(self.FirstVarName, self.SecondVarName, value, rcmdobjlist, context, varmap)
	if err != nil {
		return controlflow, err
	}
	return controlflow, nil
}

func doInCondition(firstVarname string, secndVarname string, value Void,
	rcmdobjlist []RcmdInterface, context *ReplayerContext, varmap VariableMap) (Void, *errors.Error) {

	if len(firstVarname) == 0 || len(secndVarname) == 0 || context == nil || varmap == nil {
		return nil, errors.New("invalid arguments")
	}

	switch value.(type) {
	case []Void:
		for idx, valueElem := range value.([]Void) {
			vari, err := NewVariable(secndVarname, valueElem, "")
			if err != nil {
				return nil, err
			}

			err = varmap.SetValue(vari)
			if err != nil {
				return nil, err
			}

			idxVari, err := NewVariable(firstVarname, float64(idx), "")
			if err != nil {
				return nil, err
			}

			err = varmap.SetValue(idxVari)
			if err != nil {
				return nil, err
			}

			controlflow, err := PlayRcmdList(rcmdobjlist, context)
			if err != nil {
				return nil, err
			}

			switch controlflow.(type) {
			case int:
				switch controlflow.(int) {
				case CF_BREAK:
					return nil, nil
				case CF_CONTINUE:
					continue
				case CF_RETURN:
					return controlflow, nil
				}
			}
		}
	case map[Void]Void:
		for key, value := range value.(map[Void]Void) {
			valueVari, err := NewVariable(secndVarname, value, "")
			if err != nil {
				return nil, err
			}

			err = varmap.SetValue(valueVari)
			if err != nil {
				return nil, err
			}

			keyVari, err := NewVariable(firstVarname, key, "")
			if err != nil {
				return nil, err
			}

			err = varmap.SetValue(keyVari)
			if err != nil {
				return nil, err
			}

			controlflow, err := PlayRcmdList(rcmdobjlist, context)
			if err != nil {
				return nil, err
			}

			switch controlflow.(type) {
			case int:
				switch controlflow.(int) {
				case CF_BREAK:
					return nil, nil
				case CF_CONTINUE:
					continue
				case CF_RETURN:
					return controlflow, nil
				}
			}
		}
	default:
		return nil, errors.New(fmt.Sprintf(`invalid "for in" value argument, variable(%s) must be a array or map`, reflect.TypeOf(value)))
	}

	return nil, nil
}

type ForRangeCondition struct {
	VarName      string      `@IDENT`
	RangeKeyword string      `@"range"`
	Start        *Expression `@@ ","`
	End          *Expression `@@`
	Step         *Expression `[ "," @@ ]`
}

func (self *ForRangeCondition) ToString() string {
	text := self.VarName + " " + self.RangeKeyword + " "
	if self.Start != nil {
		text += self.Start.ToString()
	}

	if self.End != nil {
		text += "," + self.End.ToString()
	}

	if self.Step != nil {
		text += "," + self.Step.ToString()
	}

	return text
}

func (self *ForRangeCondition) Do(rcmdobjlist []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	var start float64 = 0
	var end float64 = 0
	var step float64 = 1

	/* Start number 얻음
	 */
	if self.Start == nil {
		return nil, errors.New(`invalid "for range" start arguments`)
	}

	value, err := self.Start.Do(context)
	if err != nil {
		return nil, err
	}

	switch value.(type) {
	case float64:
		start = value.(float64)
	default:
		return nil, errors.New(`invalid "for range" start arguments`)
	}

	/* End number 얻음
	 */
	if self.End == nil {
		return nil, errors.New(`invalid "for range" end arguments`)
	}

	value, err = self.End.Do(context)
	if err != nil {
		return nil, err
	}

	switch value.(type) {
	case float64:
		end = value.(float64)
	default:
		return nil, errors.New(`invalid "for range" end arguments`)
	}

	/* Step number 얻음
	 */
	if self.Step != nil {
		value, err = self.Step.Do(context)
		if err != nil {
			return nil, err
		}

		switch value.(type) {
		case float64:
			step = value.(float64)
		default:
			return nil, errors.New(`invalid "for range" step arguments`)
		}
	}

	if step == 0 {
		return nil, errors.New("Invalid For range sentence, step have to not zero")
	} else if step > 0 && start > end {
		return nil, errors.New("Invalid For range sentence, start have to greater than end")
	} else if step < 0 && end > start {
		return nil, errors.New("Invalid For range sentence, end have to smaller than start")
	}

	varmap := NewVariableMap()
	context.PushVarMapSlice(varmap)
	defer context.PopVarMapSlice()

	if start <= end {
		for idx := start; idx < end; idx += step {
			var1, err1 := NewVariable(self.VarName, float64(idx), "")
			if err1 != nil {
				return nil, err1
			}

			err1 = varmap.SetValue(var1)
			if err1 != nil {
				return nil, err1
			}

			controlflow, err := PlayRcmdList(rcmdobjlist, context)
			if err != nil {
				return nil, err
			}

			switch controlflow.(type) {
			case int:
				switch controlflow.(int) {
				case CF_BREAK:
					return nil, nil
				case CF_CONTINUE:
					continue
				case CF_RETURN:
					return controlflow, nil
				}
			}
		}
	} else {
		for idx := start - 1; idx >= end; idx += step {
			var1, err1 := NewVariable(self.VarName, float64(idx), "")
			if err1 != nil {
				return nil, err1
			}

			err1 = varmap.SetValue(var1)
			if err1 != nil {
				return nil, err1
			}

			controlflow, err := PlayRcmdList(rcmdobjlist, context)
			if err != nil {
				return nil, err
			}

			switch controlflow.(type) {
			case int:
				switch controlflow.(int) {
				case CF_BREAK:
					return nil, nil
				case CF_CONTINUE:
					continue
				case CF_RETURN:
					return controlflow, nil
				}
			}
		}
	}

	return nil, nil
}

type For struct {
	Name              string             `@"for"`
	ForInCondition    *ForInCondition    `(@@`
	ForRangeCondition *ForRangeCondition `|@@)`
	RcmdList          *RcmdList          `@@`
	EndforKeyword     string             `@"endfor"`

	RcmdObjList []RcmdInterface
}

func NewFor(text string) (*For, *errors.Error) {
	target, err := NewStruct(text, &For{})
	if err != nil {
		return nil, err
	}
	return target.(*For), nil
}

func (self *For) ToString() string {
	text := self.Name + " "
	if self.ForInCondition != nil {
		text += self.ForInCondition.ToString()
	} else if self.ForRangeCondition != nil {
		text += self.ForRangeCondition.ToString()
	}
	/* XXX
	text += "\n"
	text += ToStringRcmdList(self.RcmdList)
	text += self.EndforKeyword
	*/

	return text
}

func (self *For) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *For) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	if self.ForInCondition != nil {
		res, err := self.ForInCondition.Do(self.RcmdObjList, context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		return res, nil
	} else if self.ForRangeCondition != nil {
		res, err := self.ForRangeCondition.Do(self.RcmdObjList, context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		return res, nil
	} else {
		return nil, errors.New(`invalid "for" sentence`).AddMsg(self.ToString())
	}

	return nil, nil
}

func (self *For) GetName() string {
	return self.Name
}

func (self *For) Dump() {
	repr.Println(self)
}
