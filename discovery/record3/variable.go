package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"reflect"
	"strconv"
	"strings"
)

/* 형변환 함수
 */

func ConvIntToFloat(unknownValue Void) Void {
	var floatType = reflect.TypeOf(float64(0))

	v := reflect.ValueOf(unknownValue)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return unknownValue
	}

	fv := v.Convert(floatType)
	return fv.Float()
}

/* map[Void]Void -> map[string]interface{} 형 변환
 */
func ConvVoidToStringMap(value Void) Void {
	switch value.(type) {
	case map[Void]Void:
		resMap := make(map[string]interface{})

		for k, v := range value.(map[Void]Void) {
			key := fmt.Sprintf("%v", k)
			if f, ok := k.(float64); ok {
				key = strconv.FormatFloat(f, 'f', -1, 64)
			}
			resMap[key] = ConvVoidToStringMap(v)
		}
		return resMap
	case []Void:
		ll := []interface{}{}
		for _, data := range value.([]Void) {
			ll = append(ll, ConvVoidToStringMap(data))
		}
		return ll
	default:
		return value
	}
}

type Variable struct {
	Name     string
	Value    Void
	LoadPath string
}

func NewVariable(name string, value Void, loadpath string) (*Variable, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	/* value type 검사
	 * array 타입은 []Void로 변환
	 */
	switch value.(type) {
	case bool:
	case []bool:
		list := []Void{}
		for _, data := range value.([]bool) {
			list = append(list, data)
		}
		value = list
	case string:
	case []string:
		list := []Void{}
		for _, data := range value.([]string) {
			list = append(list, data)
		}
		value = list
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32:
		value = ConvIntToFloat(value)
	case float64:
	case []float64:
		list := []Void{}
		for _, data := range value.([]float64) {
			list = append(list, data)
		}
		value = list
	case []interface{}:
		list := []Void{}
		for _, data := range value.([]interface{}) {
			vari, err := NewVariable("_", data, "")
			if err != nil {
				return nil, err
			}
			list = append(list, vari.Value)
		}
		value = list
	case []Void:
		list := []Void{}
		for _, data := range value.([]Void) {
			vari, err := NewVariable("_", data, "")
			if err != nil {
				return nil, err
			}
			list = append(list, vari.Value)
		}
		value = list
	case map[string]interface{}:
		res := make(map[Void]Void)
		for key, value := range value.(map[string]interface{}) {
			vari, err := NewVariable("_", value, "")
			if err != nil {
				return nil, err
			}
			res[key] = vari.Value
		}
		value = res
	case map[Void]Void:
		res := make(map[Void]Void)
		for key, value := range value.(map[Void]Void) {
			vari, err := NewVariable("_", value, "")
			if err != nil {
				return nil, err
			}
			res[key] = vari.Value
		}
		value = res
	case nil:
	default:
		return nil, errors.New("invalid value type")
	}

	vari := Variable{
		Name:     name,
		Value:    value,
		LoadPath: loadpath,
	}

	return &vari, nil
}

func (self *Variable) SetValue(value Void) *errors.Error {
	if len(self.LoadPath) > 0 {
		return errors.New(fmt.Sprintf("Can't set to load variable, %s variable is seted by %s", self.Name, self.LoadPath))
	}
	self.Value = value
	return nil
}

/* name - variable map
 */
type VariableMap map[string]*Variable

func NewVariableMap() VariableMap {
	return make(VariableMap)
}

func (self VariableMap) SetValue(variable *Variable) *errors.Error {
	if variable == nil {
		return errors.New("Invalid arguments")
	}

	v, _ := self.FindValue(variable.Name)
	if v == nil {
		self[variable.Name] = variable
	} else {
		/* load 된 변수는 set 못하도록 함
		 */
		if len(v.LoadPath) == 0 {
			self[variable.Name] = variable
		} else {
			return errors.New(fmt.Sprintf("Can't set to loaded variable, %s variable is seted by %s", v.Name, v.LoadPath))
		}
	}

	return nil
}

func (self VariableMap) FindValue(name string) (*Variable, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	var variable *Variable = nil
	var ok bool = false

	/* name-variable map에서 name key 값이 있는지 찾음
	 */
	variable, ok = self[name]
	if !ok {
		return nil, errors.New(fmt.Sprintf("key=%s variable not found", name))
	}

	if variable == nil {
		return nil, errors.New(fmt.Sprintf("key=%s variable is nil", name))
	}

	return variable, nil
}

func (self VariableMap) FindValueWithLoadPath(loadpath string) []*Variable {
	list := []*Variable{}

	for _, value := range self {
		if value.LoadPath == loadpath {
			list = append(list, value)
		}
	}

	return list
}

func (self VariableMap) SetTable(nameArr, valueArr []string, loadpath string) *errors.Error {
	if len(nameArr) != len(valueArr) {
		return errors.New("invalid arguments")
	}

	for idx, nameStr := range nameArr {
		name := strings.Replace(strings.TrimSpace(nameStr), " ", "_", -1)
		data := string(valueArr[idx])

		/* table은 loadpath 설정 안함
		 */
		var1, err := NewVariable(name, data, "")
		if err != nil {
			return err
		}

		err = self.SetValue(var1)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self VariableMap) DelValue(key string) *errors.Error {
	if len(key) == 0 {
		return errors.New("Invalid arguments")
	}

	delete(self, key)
	return nil
}

func (self VariableMap) DelValueWithLoadPath(loadfile string) *errors.Error {
	if len(loadfile) == 0 {
		return errors.New("Invalid arguments")
	}

	for key, var1 := range self {
		if var1.LoadPath == loadfile {
			self.DelValue(key)
		}
	}

	return nil
}
