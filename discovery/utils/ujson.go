package utils

import (
	"discovery/errors"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"reflect"
)

type Ujson struct {
	Data map[string]interface{}
}

func NewUjson(jsonInput []byte) (*Ujson, *errors.Error) {
	data := make(map[string]interface{})
	oserr := json.Unmarshal(jsonInput, &data)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	return &Ujson{Data: data}, nil
}

func NewUjsonWithPath(filepath string) (*Ujson, *errors.Error) {
	jsonData, oserr := ioutil.ReadFile(filepath)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	return NewUjson(jsonData)
}

func (self *Ujson) Get(key string) (interface{}, *errors.Error) {
	if self.Data == nil {
		return nil, errors.New("Data is nil")
	}

	value, ok := self.Data[key]
	if !ok {
		return nil, errors.New(fmt.Sprintf("'%s' key doesn't exist", key))
	}

	return value, nil
}

func (self *Ujson) GetData(key string) (map[string]interface{}, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return nil, err
	}

	return value.(map[string]interface{}), nil
}

func (self *Ujson) Next(key string) *Ujson {
	value, err := self.Get(key)
	if err != nil {
		log.Println("ERR:", err)
		return nil
	}

	j := Ujson{}
	switch value.(type) {
	case map[string]interface{}:
		j.Data = value.(map[string]interface{})
	default:
		log.Println(fmt.Sprintf("ERR: %v value is not a valid type", value))
		return nil
	}

	return &j
}

func (self *Ujson) List(key string, idx uint32) *Ujson {
	value, err := self.Get(key)
	if err != nil {
		log.Println("ERR:", err)
		return nil
	}

	j := Ujson{}
	switch value.(type) {
	case []interface{}:
		tmp := value.([]interface{})
		if idx >= uint32(len(tmp)) {
			return nil
		}
		j.Data = tmp[idx].(map[string]interface{})
	default:
		return nil
	}

	return &j
}

func (self *Ujson) Keys() []string {
	keylist := []string{}

	if self.Data != nil {
		for k, _ := range self.Data {
			keylist = append(keylist, k)
		}
	}

	return keylist
}

func (self *Ujson) Exist(key string) bool {
	_, ok := self.Data[key]
	return ok
}

func (self *Ujson) IsNil(key string) bool {
	value, err := self.Get(key)
	if err != nil {
		return true
	}

	if value == nil {
		return true
	}

	return false
}

func (self *Ujson) GetString(key string) (string, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return "", err
	}

	strValue, ok := value.(string)
	if ok {
		return strValue, nil
	}

	return "", errors.New(fmt.Sprintf("%v type is %s", value, reflect.TypeOf(value)))
}

func (self *Ujson) GetInt(key string) (int, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return 0, err
	}

	intValue, ok := value.(float64)
	if ok {
		return int(intValue), nil
	}

	return 0, errors.New(fmt.Sprintf("%v type is %s", value, reflect.TypeOf(value)))
}

func (self *Ujson) GetBool(key string) (bool, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return false, err
	}

	boolValue, ok := value.(bool)
	if ok {
		return boolValue, nil
	}

	return false, errors.New(fmt.Sprintf("%v type is %s", value, reflect.TypeOf(value)))
}

func (self *Ujson) GetList(key string) ([]string, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return []string{}, err
	}

	list, ok := value.([]interface{})
	if ok {
		strList := []string{}
		for _, element := range list {
			strList = append(strList, fmt.Sprintf("%v", element))
		}

		return strList, nil
	}

	return []string{}, errors.New(fmt.Sprintf("%v type is %s", value, reflect.TypeOf(value)))
}

func (self *Ujson) GetJson(key string) (string, *errors.Error) {
	value, err := self.Get(key)
	if err != nil {
		return "", err
	}

	jsonData, oserr := json.Marshal(value)
	if oserr != nil {
		return "", errors.New(fmt.Sprintf("%s", oserr))
	}

	return string(jsonData), nil
}
