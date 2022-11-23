package record3

import (
	"discovery/errors"
	"discovery/utils"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

/* Void 정의
 */
type Void interface{}

const (
	EMPTY int = iota
	T_BOOL
	T_NUMBER
	T_STRING
	T_RSTRING
	T_STRING_LIST
	T_STRING_MAP
	T_VARIABLE
)

/* parimary index 처리, array: index, slice
 * map key 처리
 */
type PrimIndex struct {
	Start     *Expression `[@@]` // array: index, slice start, map의 key
	SliceFlag bool        `[@(","|":")`
	End       *Expression ` [@@]]`
}

func (self *PrimIndex) ToString() string {
	text := ""
	if self.Start != nil {
		text += self.Start.ToString()
	}

	if self.SliceFlag {
		text += ","
	}

	if self.End != nil {
		text += self.End.ToString()
	}

	return text
}

func (self *PrimIndex) Do(value Void, context *ReplayerContext) (Void, *errors.Error) {
	switch value.(type) {
	case []Void, string, []interface{}:
		size := int32(0)
		switch value.(type) {
		case []Void:
			size = int32(len(value.([]Void)))
		case []interface{}:
			size = int32(len(value.([]interface{})))
		case string:
			size = int32(len(value.(string)))
		default:
			return nil, errors.New("value is not a array")
		}

		if self.SliceFlag {
			start := int32(0)

			if self.Start != nil {
				v, err := self.Start.Do(context)
				if err != nil {
					return nil, err
				}

				switch v.(type) {
				case float64:
					start = int32(v.(float64))
				default:
					return nil, errors.New("index value is not a numeric")
				}

				if start < 0 {
					return nil, errors.New("index is out of range")
				}
			}

			end := size
			if self.End != nil {
				v, err := self.End.Do(context)
				if err != nil {
					return nil, err
				}

				switch v.(type) {
				case float64:
					end = int32(v.(float64))
				default:
					return nil, errors.New("index value is not a numeric")
				}

				if end > size {
					return nil, errors.New("index is out of range")
				}
			}

			switch value.(type) {
			case []Void:
				return value.([]Void)[start:end], nil
			case []interface{}:
				return value.([]interface{})[start:end], nil
			case string:
				return value.(string)[start:end], nil
			}
			return nil, errors.New("value is not a array")
		} else {
			if self.Start == nil || self.End != nil {
				return nil, errors.New("invalid array index arguments")
			}

			idx := int32(0)
			v, err := self.Start.Do(context)
			if err != nil {
				return nil, err
			}

			switch v.(type) {
			case float64:
				idx = int32(v.(float64))
			default:
				return nil, errors.New("index value is not a numeric")
			}

			if idx < 0 || idx >= size {
				return nil, errors.New("index is out of range")
			}

			switch value.(type) {
			case []Void:
				return value.([]Void)[idx], nil
			case []interface{}:
				return value.([]interface{})[idx], nil
			case string:
				return string(value.(string)[idx]), nil
			}
			return nil, errors.New("value is not a array")
		}
	case map[Void]Void, map[string]interface{}:
		if self.SliceFlag {
			return nil, errors.New("invalid map key arguments")
		}

		if self.Start == nil || self.End != nil {
			return nil, errors.New("invalid array key arguments")
		}

		v, err := self.Start.Do(context)
		if err != nil {
			return nil, err
		}

		switch value.(type) {
		case map[Void]Void:
			res, ok := value.(map[Void]Void)[v]
			if !ok {
				return nil, errors.New(fmt.Sprintf("'%v' map key is invalid", v))
			}
			return res, nil
		case map[string]interface{}:
			res, ok := value.(map[string]interface{})[v.(string)]
			if !ok {
				return nil, errors.New(fmt.Sprintf("'%v' map key is invalid", v))
			}
			return res, nil
		}
	}

	return nil, errors.New(fmt.Sprintf("value(%s) is not a array or map", reflect.TypeOf(value)))
}

/* value가 function pointer 일때 function parameter
 */
type FuncParam struct {
	Parameters []*Expression `[ @@ { "," @@ } ]`
}

func (self *FuncParam) ToString() string {
	text := ""

	for i, expr := range self.Parameters {
		if i == 0 {
			text = expr.ToString()
		} else {
			text += "," + expr.ToString()
		}
	}

	return text
}

func (self *FuncParam) Do(value Void, context *ReplayerContext) (Void, *errors.Error) {
	var f FunctionInterface

	switch value.(type) {
	case FunctionInterface:
		f = value.(FunctionInterface)
	default:
		return nil, errors.New("Value is not a function ")
	}

	res := f.Do(context, self.Parameters)
	switch res.(type) {
	case *errors.Error:
		return nil, res.(*errors.Error)
	case *Expression:
		value2, err := res.(*Expression).Do(context)
		if err != nil {
			return nil, err
		}
		return value2, nil
	}
	return res, nil
}

type PrimParam struct {
	Idx       *PrimIndex `( "[" @@ "]" `  // array, map index
	FuncParam *FuncParam ` |"(" @@ ")" )` // function parameter
	Next      *PrimParam `{ @@ } `
}

func (self *PrimParam) ToString() string {
	text := ""

	if self.Idx != nil {
		text += "[" + self.Idx.ToString() + "]"
	} else if self.FuncParam != nil {
		text += "(" + self.FuncParam.ToString() + ")"
	}

	if self.Next != nil {
		text += self.Next.ToString()
	}

	return text
}

func (self *PrimParam) Do(value Void, context *ReplayerContext) (Void, *errors.Error) {
	var resValue Void

	if self.Idx != nil {
		res, err := self.Idx.Do(value, context)
		if err != nil {
			return nil, err
		}
		resValue = res
	} else if self.FuncParam != nil {
		res, err := self.FuncParam.Do(value, context)
		if err != nil {
			return nil, err
		}
		resValue = res
	} else {
		return nil, errors.New("invalid PrimParam parameter")
	}

	if self.Next != nil {
		return self.Next.Do(resValue, context)
	}
	return resValue, nil
}

/* array 나 map 요소(index, key)에 값 설정
 */
func (self *PrimParam) SetParamValue(value *Void, newValue Void, context *ReplayerContext) *errors.Error {
	if value == nil || context == nil {
		return errors.New("invalid argument")
	}

	if self.FuncParam != nil {
		return errors.New("invalid syntax")
	}

	switch (*value).(type) {
	case bool, string, float64:
		if self.Idx != nil {
			return errors.New("variable is not a array or map")
		}

		*value = newValue
		return nil

	case []Void, map[Void]Void:
		if self.Idx == nil {
			*value = newValue
			return nil
		}

		if self.Idx.Start == nil || self.Idx.SliceFlag || self.Idx.End != nil {
			return errors.New("invalid syntax")
		}
		idxExpr := self.Idx.Start

		idxkey, err := idxExpr.Do(context)
		if err != nil {
			return err
		}

		switch (*value).(type) {
		case []Void:
			f, ok := idxkey.(float64)
			if !ok {
				return errors.New("index is not a integer")
			}
			idx := int(f)

			if idx < 0 || idx >= len((*value).([]Void)) {
				return errors.New("index is out of range")
			}

			if self.Next != nil {
				elem := (*value).([]Void)[idx]
				err := self.Next.SetParamValue(&elem, newValue, context)
				if err != nil {
					return err
				}
				(*value).([]Void)[idx] = elem
			} else {
				(*value).([]Void)[idx] = newValue
			}
		case map[Void]Void:
			if self.Next != nil {
				elem, ok := (*value).(map[Void]Void)[idxkey]
				if ok {
					err := self.Next.SetParamValue(&elem, newValue, context)
					if err != nil {
						return err
					}
					(*value).(map[Void]Void)[idxkey] = elem
				} else {
					(*value).(map[Void]Void)[idxkey] = newValue
				}
			} else {
				(*value).(map[Void]Void)[idxkey] = newValue
			}
		}
		return nil
	}

	return errors.New(fmt.Sprintf("'%s' unknown value type", reflect.TypeOf(value)))
}

/* array 나 map 요소(index, key) 값 삭제
 */
func (self *PrimParam) DeleteParamValue(value *Void, context *ReplayerContext) *errors.Error {
	if value == nil || context == nil {
		return errors.New("invalid argument")
	}

	if self.FuncParam != nil {
		return errors.New("invalid syntax")
	}

	switch (*value).(type) {
	case bool, string, float64:
		if self.Idx != nil {
			return errors.New("variable is not a array or map")
		}
		return nil

	case []Void:
		if self.Idx == nil {
			return nil
		}
		primIndex := self.Idx

		if self.Next == nil {
			size := int32(len((*value).([]Void)))

			if primIndex.SliceFlag {
				start := int32(0)
				if primIndex.Start != nil {
					v, err := primIndex.Start.Do(context)
					if err != nil {
						return err
					}

					switch v.(type) {
					case float64:
						start = int32(v.(float64))
					default:
						return errors.New("index value is not a numeric")
					}

					if start < 0 {
						return errors.New("index is out of range")
					}
				}

				end := size
				if primIndex.End != nil {
					v, err := primIndex.End.Do(context)
					if err != nil {
						return err
					}

					switch v.(type) {
					case float64:
						end = int32(v.(float64))
					default:
						return errors.New("index value is not a numeric")
					}
					if end > size {
						return errors.New("index is out of range")
					}
				}

				if start > end {
					return errors.New("index is out of range")
				}

				/* array value값 삭제
				 */
				*value = append((*value).([]Void)[:start], (*value).([]Void)[end:]...)
				return nil
			}

			/* slice flag 없을 경우
			 */
			if primIndex.Start == nil || primIndex.End != nil {
				return errors.New("invalid array index arguments")
			}

			idx := int32(0)
			v, err := primIndex.Start.Do(context)
			if err != nil {
				return err
			}

			switch v.(type) {
			case float64:
				idx = int32(v.(float64))
			default:
				return errors.New("index value is not a numeric")
			}

			if idx < 0 || idx >= size {
				return errors.New("index is out of range")
			}

			/* array value값 삭제
			 */
			*value = append((*value).([]Void)[:idx], (*value).([]Void)[idx+1:]...)
			return nil
		}

		/* next가 있는 경우
		 */
		if primIndex.Start == nil || primIndex.SliceFlag || primIndex.End != nil {
			return errors.New("invalid syntax")
		}

		idxValue, err := primIndex.Start.Do(context)
		if err != nil {
			return err
		}

		f, ok := idxValue.(float64)
		if !ok {
			return errors.New("index is not a integer")
		}
		idx := int(f)

		if idx < 0 || idx >= len((*value).([]Void)) {
			return errors.New("index is out of range")
		}

		elem := (*value).([]Void)[idx]
		err = self.Next.DeleteParamValue(&elem, context)
		if err != nil {
			return err
		}

		(*value).([]Void)[idx] = elem
		return nil

	case map[Void]Void:
		if self.Idx == nil {
			return nil
		}
		primIndex := self.Idx

		/* map은 slice flag가 없어야 함
		 */
		if primIndex.Start == nil || primIndex.SliceFlag || primIndex.End != nil {
			return errors.New("invalid unset map index arguments")
		}

		key, err := primIndex.Start.Do(context)
		if err != nil {
			return err
		}

		if self.Next == nil {
			/* next map index가 없으면 delete
			 */
			delete((*value).(map[Void]Void), key)
		} else {
			elem, ok := (*value).(map[Void]Void)[key]
			if !ok {
				return errors.New(fmt.Sprintf("'%v' map key is invalid", key))
			}

			err := self.Next.DeleteParamValue(&elem, context)
			if err != nil {
				return err
			}
			(*value).(map[Void]Void)[key] = elem
		}
		return nil
	}

	return errors.New(fmt.Sprintf("'%s' unknown value type", reflect.TypeOf(value)))
}

/* Map 자료구조의 key:value
 */
type MapKeyvalue struct {
	Key   *Expression `@@ ":"`
	Value *Expression `@@`
}

func NewMapKeyvalue(key string, value string) (*MapKeyvalue, *errors.Error) {
	keyExp, err := NewExpression(key)
	if err != nil {
		return nil, err
	}

	valueExp, err := NewExpression(value)
	if err != nil {
		return nil, err
	}

	m := MapKeyvalue{
		Key:   keyExp,
		Value: valueExp,
	}

	return &m, nil
}

func (self *MapKeyvalue) ToString() string {
	text := ""

	if self.Key != nil {
		text += self.Key.ToString() + ":"
	}

	if self.Value != nil {
		text += self.Value.ToString()
	}

	return text
}

func (self *MapKeyvalue) Do(context *ReplayerContext) (Void, Void, *errors.Error) {
	if self.Key == nil {
		return nil, nil, errors.New("invalid map key parameter")

	}

	if self.Value == nil {
		return nil, nil, errors.New("invalid map value parameter")
	}

	key, err := self.Key.Do(context)
	if err != nil {
		return nil, nil, err
	}

	value, err := self.Value.Do(context)
	if err != nil {
		return nil, nil, err
	}

	return key, value, nil
}

/* primary 정의
 */
type Primary struct {
	Bool          *string        `@("true"|"on"|"false"|"off")`
	Null          *string        `|@("null"|"nil")`
	Number        *float64       `|@NUMBER`
	String        *string        `|@STRING`
	RString       *string        `|"r" @STRING`
	Variable      *string        `|@(IDENT|FUNCTION)`
	ListFlag      bool           `|@"["`
	List          []*Expression  `     [ @@ { "," @@ } ] "]"`
	MapFlag       bool           `|@"{"`
	Map           []*MapKeyvalue `     [ @@ { "," @@ } ] "}"`
	SubExpression *Expression    `|"(" @@ ")" `
}

func NewPrimary(text string) (*Primary, *errors.Error) {
	prim, err := NewStruct(text, &Primary{})
	if err != nil {
		return nil, err
	}
	return prim.(*Primary), nil
}

func NewPrimaryWithType(primType int, value Void) (*Primary, *errors.Error) {
	prim := Primary{}

	switch primType {
	case T_BOOL:
		tmp := value.(string)
		switch strings.ToLower(tmp) {
		case "true", "on":
		case "false", "off":
		default:
			return nil, errors.New("invalid bool string")
		}
		prim.Bool = &tmp
	case T_NUMBER:
		num := value.(float64)
		prim.Number = &num
	case T_STRING:
		str := value.(string)
		str = utils.Quote(str)
		prim.String = &str
	case T_RSTRING:
		str := value.(string)
		str = utils.Quote(str)
		prim.RString = &str
	case T_VARIABLE:
		str := value.(string)

		re, oserr := regexp.Compile(`^[a-zA-Z_][a-zA-Z0-9_:]*$`)
		if oserr != nil {
			return nil, errors.New(fmt.Sprintf("%s", oserr))
		}

		if re.MatchString(str) {
			prim.Variable = &str
		} else {
			return nil, errors.New(fmt.Sprintf("'%v' is invalid variable name", str))
		}
	case T_STRING_LIST:
		prim.ListFlag = true
		for _, str := range value.([]string) {
			e, err := NewExpression(utils.Quote(str))
			if err != nil {
				return nil, err
			}
			prim.List = append(prim.List, e)
		}
	case T_STRING_MAP:
		prim.MapFlag = true
		for key, value := range value.(map[string]string) {
			e, err := NewMapKeyvalue(utils.Quote(key), utils.Quote(value))
			if err != nil {
				return nil, err
			}
			prim.Map = append(prim.Map, e)
		}
	default:
		return nil, errors.New("invalid arguments")
	}

	return &prim, nil
}

func (self *Primary) ToString() string {
	if self.Bool != nil {
		return *self.Bool
	} else if self.Null != nil {
		return fmt.Sprintf("null")
	} else if self.Number != nil {
		return strconv.FormatFloat(*self.Number, 'f', -1, 64)
	} else if self.String != nil {
		return fmt.Sprintf("%s", utils.Quote(utils.Unquote(*self.String)))
	} else if self.RString != nil {
		return fmt.Sprintf("r%s", utils.Quote(utils.Unquote(*self.RString)))
	} else if self.Variable != nil {
		return *self.Variable
	} else if self.ListFlag {
		text := "["
		for i, prim := range self.List {
			if i > 0 {
				text += ", "
			}
			text += prim.ToString()
		}
		text += "]"
		return text
	} else if self.MapFlag {
		text := "{"
		for i, kv := range self.Map {
			if i > 0 {
				text += ", "
			}
			text += kv.ToString()
		}
		text += "}"
		return text
	} else if self.SubExpression != nil {
		return "(" + self.SubExpression.ToString() + ")"
	}

	return ""
}

func (self *Primary) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Bool != nil {
		switch strings.ToLower(*self.Bool) {
		case "true", "on":
			return true, nil
		case "false", "off":
			return false, nil
		}
	} else if self.Null != nil {
		return nil, nil
	} else if self.Number != nil {
		return *self.Number, nil
	} else if self.String != nil {
		out, err := context.ReplaceVariable(utils.Unquote(*self.String))
		if err != nil {
			return nil, err
		}
		return out, nil
	} else if self.RString != nil {
		out, err := context.ReplaceVariable(utils.Unquote(*self.RString))
		if err != nil {
			return nil, err
		}
		re, oserr := regexp.Compile(out)
		if oserr != nil {
			return nil, errors.New(fmt.Sprintf("%s", oserr))
		}
		return re, nil
	} else if self.Variable != nil {
		value2, ok := context.FunctionMap[*self.Variable]
		if !ok {
			value, err := context.GetVariableValue(*self.Variable)
			if err != nil {
				return nil, err
			}
			return value, nil
		}
		return value2, nil
	} else if self.ListFlag {
		var list []Void
		for _, pri := range self.List {
			res, err := pri.Do(context)
			if err != nil {
				return nil, err
			}
			list = append(list, res)
		}
		return list, nil
	} else if self.MapFlag {
		mapdata := make(map[Void]Void)
		for _, kv := range self.Map {
			key, value, err := kv.Do(context)
			if err != nil {
				return nil, err
			}

			if _, ok := mapdata[key]; ok {
				return nil, errors.New(fmt.Sprintf("'%v' map key already exist", key))
			}

			mapdata[key] = value
		}
		return mapdata, nil
	} else if self.SubExpression != nil {
		return self.SubExpression.Do(context)
	}

	return nil, errors.New("unknown primary")
}

func (self *Primary) GetBool() (bool, *errors.Error) {
	if self.Bool != nil {
		switch strings.ToLower(*self.Bool) {
		case "true", "on":
			return true, nil
		case "false", "off":
			return false, nil
		}
	}
	return false, errors.New("empty")
}

func (self *Primary) GetNumber() (float64, *errors.Error) {
	if self.Number != nil {
		return *self.Number, nil
	}
	return 0, errors.New("empty")
}

func (self *Primary) GetString() (string, *errors.Error) {
	if self.String != nil {
		return utils.Unquote(*self.String), nil
	}
	return "", errors.New("empty")
}

func (self *Primary) GetRString() (*regexp.Regexp, string, *errors.Error) {
	if self.RString != nil {
		rstring := utils.Unquote(*self.RString)
		re, oserr := regexp.Compile(rstring)
		if oserr != nil {
			return nil, rstring, errors.New(fmt.Sprintf("%s", oserr))
		}
		return re, rstring, nil
	}

	return nil, "", errors.New("empty")
}

func (self *Primary) GetVariable() (string, *errors.Error) {
	if self.Variable != nil {
		return *self.Variable, nil
	}

	return "", errors.New("empty")
}

/* Primary에서 List 변수 핸들
 * 거의 사용하지 않음
 */
func (self *Primary) GetList() ([]*Expression, *errors.Error) {
	if self.ListFlag {
		return self.List, nil
	}
	return []*Expression{}, errors.New("empty")
}

func (self *Primary) GetListSize() (uint32, *errors.Error) {
	if self.ListFlag {
		return uint32(len(self.List)), nil
	}
	return 0, errors.New("empty")
}

func (self *Primary) GetListElem(idx uint32) (*Expression, *errors.Error) {
	if self.ListFlag {
		if idx >= uint32(len(self.List)) {
			return nil, errors.New("index out of range")
		}
		return self.List[idx], nil
	}
	return nil, errors.New("empty")
}

/* Primary에서 Map 변수 핸들
 * 거의 사용하지 않음
 */
func (self *Primary) GetMap() ([]*MapKeyvalue, *errors.Error) {
	if self.MapFlag {
		return self.Map, nil
	}
	return []*MapKeyvalue{}, errors.New("empty")
}

func (self *Primary) GetMapSize() (uint32, *errors.Error) {
	if self.MapFlag {
		return uint32(len(self.Map)), nil
	}
	return 0, errors.New("empty")
}

func (self *Primary) GetMapElem(idx uint32) (*MapKeyvalue, *errors.Error) {
	if self.MapFlag {
		if idx >= uint32(len(self.Map)) {
			return nil, errors.New("index out of range")
		}
		return self.Map[idx], nil
	}
	return nil, errors.New("empty")
}

/* 변수, 배열 변수
 * 함수
 */
type PrimValue struct {
	Value *Primary   `@@`     // 변수, 함수 이름
	Param *PrimParam `[ @@ ]` // array 변수 index, function parameter
}

func NewPrimValue(text string) (*PrimValue, *errors.Error) {
	primvari, err := NewStruct(text, &PrimValue{})
	if err != nil {
		return nil, err
	}
	return primvari.(*PrimValue), nil
}

func (self *PrimValue) ToString() string {
	text := ""

	if self.Value != nil {
		text += self.Value.ToString()
	}

	if self.Param != nil {
		text += self.Param.ToString()
	}

	return text
}

func (self *PrimValue) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Value == nil {
		return nil, errors.New("invalid primary value")
	}

	value, err := self.Value.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Param != nil {
		return self.Param.Do(value, context)
	} else {
		switch value.(type) {
		case FunctionInterface:
			return nil, errors.New("function needs parameters")
		}
	}
	return value, nil
}
