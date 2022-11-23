package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

type FunctionInterface interface {
	Do(context *ReplayerContext, parameters []*Expression) Void
}

func NewFunctionList() (map[string]FunctionInterface, *errors.Error) {
	list := make(map[string]FunctionInterface)

	list["len"] = &FuncLen{}
	list["num"] = &FuncNum{}
	list["str"] = &FuncStr{}
	list["expr"] = &FuncExpr{}
	list["split"] = &FuncSplit{}
	list["join"] = &FuncJoin{}
	list["trim"] = &FuncTrim{}
	list["filter"] = &FuncFilter{}
	list["type"] = &FuncType{}
	list["append"] = &FuncAppend{}
	list["isdefined"] = &FuncIsdefined{}

	return list, nil
}

func getParamArgs(context *ReplayerContext, parameters []*Expression) ([]Void, *errors.Error) {
	var args []Void

	for _, expr := range parameters {
		res, err := expr.Do(context)
		if err != nil {
			return args, err
		}
		args = append(args, res)
	}
	return args, nil
}

/* len 내부 함수 대응
 */
type FuncLen struct{}

func (self *FuncLen) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("invalid arguments")
	}

	first := n[0]
	switch first.(type) {
	case []interface{}:
		return float64(len(first.([]interface{})))
	case map[Void]Void:
		return float64(len(first.(map[Void]Void)))
	case map[string]interface{}:
		return float64(len(first.(map[string]interface{})))
	case []Void:
		return float64(len(first.([]Void)))
	case []string:
		return float64(len(first.([]string)))
	case []float64:
		return float64(len(first.([]float64)))
	case []bool:
		return float64(len(first.([]bool)))
	case string:
		return float64(len(first.(string)))
	default:
		return float64(0)
	}
}

/* typecast string to number
 */
type FuncNum struct{}

func (self *FuncNum) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("invalid arguments")
	}

	first := n[0]
	switch first.(type) {
	case string:
		f, oserr := strconv.ParseFloat(first.(string), 64)
		if oserr != nil {
			return errors.New(fmt.Sprintf("%s", oserr))
		}
		return float64(f)
	case float64:
		return first
	default:
		return errors.New("invalid arguments")
	}
}

/* typecast number to string
 */
type FuncStr struct{}

func (self *FuncStr) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("invalid arguments")
	}

	first := n[0]
	if f, ok := first.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64)
	} else {
		return fmt.Sprintf("%v", first)
	}
}

/* check file exist
 */
type FuncExist struct{}

func (self *FuncExist) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("exist function, invalid arguments")
	}

	first := n[0]
	switch first.(type) {
	case string:
	default:
		return errors.New("exist function, invalid arguments")
	}

	_, oserr := os.Stat(first.(string))
	if oserr == nil {
		return true
	} else if os.IsNotExist(oserr) {
		return false
	} else {
		return errors.New(fmt.Sprintf("%s", oserr))
	}
}

/* check file exist
 */
type FuncExpr struct{}

func (self *FuncExpr) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("expr function, invalid arguments")
	}

	first := n[0]
	switch first.(type) {
	case string:
	default:
		return errors.New("expr function, first arguments have to be string")
	}

	expr, err := NewExpression(first.(string))
	if err != nil {
		return err
	}

	return expr
}

/* string split
 */
type FuncSplit struct{}

func (self *FuncSplit) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 3 {
		return errors.New("split function, invalid arguments")
	}

	var stringData string
	switch n[0].(type) {
	case string:
		stringData = n[0].(string)
	default:
		return errors.New("split function, first argument have to be string")
	}

	var splitCount int
	switch n[2].(type) {
	case float64:
		splitCount = int(n[2].(float64))
	case int:
		splitCount = n[2].(int)
	default:
		return errors.New("split function, third argument have to be numeric")
	}

	var output []string

	/* split 구분자가 string 인 경우, regexp 인 경우 구분
	 */
	switch n[1].(type) {
	case string:
		sep := n[1].(string)
		output = strings.SplitN(stringData, sep, splitCount)
	case *regexp.Regexp:
		re := n[1].(*regexp.Regexp)
		output = re.Split(stringData, splitCount)
	default:
		return errors.New("split function, second argument have to be string or regexp")
	}

	out2 := []Void{}
	for _, e := range output {
		out2 = append(out2, e)
	}
	return out2
}

/* array String join
 */
type FuncJoin struct{}

func (self *FuncJoin) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 2 {
		return errors.New("join function, invalid arguments")
	}

	second := n[1]
	switch second.(type) {
	case string:
	default:
		return errors.New("join function, second argument have to be string")
	}

	first := n[0]
	out2 := []string{}

	switch first.(type) {
	case []Void:
		for _, e := range first.([]Void) {
			if f, ok := e.(float64); ok {
				out2 = append(out2, strconv.FormatFloat(f, 'f', -1, 64))
			} else {
				out2 = append(out2, fmt.Sprintf("%v", e))
			}
		}
	case []string:
		out2 = first.([]string)
	default:
		return errors.New("join function, first argument have to be []Void or []string")
	}

	return strings.Join(out2, second.(string))
}

/* space trim
 */
type FuncTrim struct{}

func (self *FuncTrim) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("trim function, invalid arguments")
	}

	first := n[0]
	var list []Void

	switch first.(type) {
	case string:
		return strings.TrimSpace(first.(string))
	case []Void:
		for _, e := range first.([]Void) {
			switch e.(type) {
			case string:
				list = append(list, strings.TrimSpace(e.(string)))
			default:
				list = append(list, e)
			}
		}
	case []string:
		for _, e := range first.([]string) {
			list = append(list, strings.TrimSpace(e))
		}
	default:
		return errors.New("trim function, first argument have to be string, []Void or []string")
	}
	return list
}

/* array, map에서 조건에 맞는 문자열 filting 후 array 리턴
 */
type FuncFilter struct{}

/* filter(array, "==", "string")
 * filter(map, "==", "id", true)
 * filter(map, "=~", r"[A-Z]*")
 */
func (self *FuncFilter) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	keyFilterFlag := false
	switch len(n) {
	case 3:
	case 4:
		flag, ok := n[3].(bool)
		if !ok {
			return errors.New("filter function, invalid key filter flag arguments")
		}
		keyFilterFlag = flag
	default:
		return errors.New("filter function, invalid arguments")
	}

	op, ok := n[1].(string)
	if !ok {
		return errors.New("filter function, invalid arguments")
	}

	res, err := doFilter(n[0], op, n[2], keyFilterFlag)
	if err != nil {
		return err
	}
	return res
}

/* map, array filter 함수
 * recursive function
 */
func doFilter(lvalue Void, op string, rvalue Void, keyFilterFlag bool) (Void, *errors.Error) {
	var output []Void

	switch op {
	case "==":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsString(rvalue) && IsString(key)) && (rvalue.(string) == key.(string)) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsString(lvalue) && IsString(rvalue) {
				if lvalue.(string) == rvalue.(string) {
					output = append(output, lvalue)
				}
			} else if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) == rvalue.(float64) {
					output = append(output, lvalue)
				}
			} else if IsBool(lvalue) && IsBool(rvalue) {
				if lvalue.(bool) == rvalue.(bool) {
					output = append(output, lvalue)
				}
			}
		}
	case "!=":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsString(rvalue) && IsString(key)) && (rvalue.(string) == key.(string)) {
						// 제외
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsString(lvalue) && IsString(rvalue) {
				if lvalue.(string) != rvalue.(string) {
					output = append(output, lvalue)
				}
			} else if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) != rvalue.(float64) {
					output = append(output, lvalue)
				}
			} else if IsBool(lvalue) && IsBool(rvalue) {
				if lvalue.(bool) != rvalue.(bool) {
					output = append(output, lvalue)
				}
			} else {
				output = append(output, lvalue)
			}
		}
	case "=~":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsString(rvalue) && IsString(key)) && strings.Contains(key.(string), rvalue.(string)) {
						output = append(output, value)
					} else if (IsRegexp(rvalue) && IsString(key)) && rvalue.(*regexp.Regexp).MatchString(key.(string)) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsString(lvalue) && IsString(rvalue) {
				if strings.Contains(lvalue.(string), rvalue.(string)) {
					output = append(output, lvalue)
				}
			} else if IsString(lvalue) && IsRegexp(rvalue) {
				if rvalue.(*regexp.Regexp).MatchString(lvalue.(string)) {
					output = append(output, lvalue)
				}
			}
		}
	case "!~":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsString(rvalue) && IsString(key)) && strings.Contains(key.(string), rvalue.(string)) {
						// 제외
					} else if (IsRegexp(rvalue) && IsString(key)) && rvalue.(*regexp.Regexp).MatchString(key.(string)) {
						// 제외
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsString(lvalue) && IsString(rvalue) {
				if !strings.Contains(lvalue.(string), rvalue.(string)) {
					output = append(output, lvalue)
				}
			} else if IsString(lvalue) && IsRegexp(rvalue) {
				if !rvalue.(*regexp.Regexp).MatchString(lvalue.(string)) {
					output = append(output, lvalue)
				}
			} else {
				output = append(output, lvalue)
			}
		}
	case ">=":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsNumeric(key) && IsNumeric(rvalue)) && key.(float64) >= rvalue.(float64) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) >= rvalue.(float64) {
					output = append(output, lvalue)
				}
			}
		}
	case ">":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsNumeric(key) && IsNumeric(rvalue)) && key.(float64) > rvalue.(float64) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) > rvalue.(float64) {
					output = append(output, lvalue)
				}
			}
		}
	case "<=":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsNumeric(key) && IsNumeric(rvalue)) && key.(float64) <= rvalue.(float64) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) <= rvalue.(float64) {
					output = append(output, lvalue)
				}
			}
		}
	case "<":
		if IsList(lvalue) {
			for _, value := range lvalue.([]Void) {
				res, err := doFilter(value, op, rvalue, keyFilterFlag)
				if err != nil {
					return nil, err
				}
				output = append(output, res.([]Void)...)
			}
		} else if IsMap(lvalue) {
			for key, value := range lvalue.(map[Void]Void) {
				if keyFilterFlag {
					if (IsNumeric(key) && IsNumeric(rvalue)) && key.(float64) < rvalue.(float64) {
						output = append(output, value)
					} else {
						res, err := doFilter(value, op, rvalue, keyFilterFlag)
						if err != nil {
							return nil, err
						}
						output = append(output, res.([]Void)...)
					}
				} else {
					res, err := doFilter(value, op, rvalue, keyFilterFlag)
					if err != nil {
						return nil, err
					}
					output = append(output, res.([]Void)...)
				}
			}
		} else if keyFilterFlag == false {
			if IsNumeric(lvalue) && IsNumeric(rvalue) {
				if lvalue.(float64) < rvalue.(float64) {
					output = append(output, lvalue)
				}
			}
		}
	default:
		return nil, errors.New("filter function, invalid operation, op have to be one of ==, !=, =~, !~, >=, >, <=, <")
	}
	return output, nil
}

/* 변수의 자료구조 문자열 출력
 */
type FuncType struct{}

func (self *FuncType) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 1 {
		return errors.New("type function, invalid arguments")
	}

	typename := reflect.TypeOf(n[0]).String()
	if typename == "string" {
		return "string"
	} else if typename == "float64" || typename == "int" {
		return "number"
	} else if strings.HasPrefix(typename, "[]") {
		return "array"
	} else if strings.HasPrefix(typename, "map") {
		return "map"
	}

	return typename
}

/* 변수의 자료구조 문자열 출력
 */
type FuncAppend struct{}

func (self *FuncAppend) Do(context *ReplayerContext, parameters []*Expression) Void {
	n, err := getParamArgs(context, parameters)
	if err != nil {
		return err
	}

	if len(n) != 2 {
		return errors.New("append function, invalid arguments")
	}

	if !IsList(n[0]) {
		return errors.New("append function, first argument have to be array variable")
	}

	return append(n[0].([]Void), n[1])
}

/* 변수의 자료구조 문자열 출력
 */
type FuncIsdefined struct{}

func (self *FuncIsdefined) Do(context *ReplayerContext, parameters []*Expression) Void {
	_, err := getParamArgs(context, parameters)
	if err != nil {
		return false
	}

	return true
}
