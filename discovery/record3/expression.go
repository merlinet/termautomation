package record3

import (
	"discovery/errors"
	"regexp"
	"strings"
)

/* 숫자, 문자열 연산 자료구조
 */
type Expression struct {
	Logical *Logical `@@`
}

func NewExpression(text string) (*Expression, *errors.Error) {
	expr, err := NewStruct(text, &Expression{})
	if err != nil {
		return nil, err
	}
	return expr.(*Expression), nil
}

func (self *Expression) ToString() string {
	if self.Logical == nil {
		return ""
	}
	return self.Logical.ToString()
}

func (self *Expression) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Logical == nil {
		return nil, errors.New("Logical is nil, invalid expression")
	}

	return self.Logical.Do(context)
}

type Logical struct {
	Equality *Equality `@@`
	Op       string    `[@("AND" | "OR")`
	Next     *Logical  ` @@]`
}

func (self *Logical) ToString() string {
	if self.Equality == nil {
		return ""
	}

	text := self.Equality.ToString()

	if self.Next == nil {
		return text
	}

	text += " " + self.Op
	text += " " + self.Next.ToString()

	return text
}

func (self *Logical) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Equality == nil {
		return nil, errors.New("Equality is nil, invalid expression")
	}

	LPri, err := self.Equality.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Next == nil {
		return LPri, nil
	}

	switch strings.ToUpper(self.Op) {
	case "AND":
		if !IsBool(LPri) {
			return nil, errors.New("AND logical operator can compare each bool value")
		}

		/* AND는 순차적으로 참인 경우 다음 비교
		 */
		if LPri.(bool) == false {
			return false, nil
		}

		RPri, err := self.Next.Do(context)
		if err != nil {
			return nil, err
		}

		if IsBool(RPri) {
			return RPri.(bool), nil
		} else {
			return nil, errors.New("AND logical operator can compare each bool value")
		}
	case "OR":
		if !IsBool(LPri) {
			return nil, errors.New("OR logical operator can compare each bool value")
		}

		/* OR는 순차적으로 참인 경우 다음 비교
		 */
		if LPri.(bool) == true {
			return true, nil
		}

		RPri, err := self.Next.Do(context)
		if err != nil {
			return nil, err
		}

		if IsBool(RPri) {
			return RPri.(bool), nil
		} else {
			return nil, errors.New("OR logical operator can compare each bool value")
		}
	default:
		return nil, errors.New("invalid logical op")
	}
}

type Equality struct {
	Comparison *Comparison `@@`
	Op         string      `[@("!" "=" | "=" "=" | "!" "~" | "=" "~")`
	Next       *Equality   ` @@]`
}

func (self *Equality) ToString() string {
	if self.Comparison == nil {
		return ""
	}

	text := self.Comparison.ToString()

	if self.Next == nil {
		return text
	}

	text += " " + self.Op
	text += " " + self.Next.ToString()

	return text
}

func (self *Equality) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Comparison == nil {
		return nil, errors.New("Comparison is nil, invalid expression")
	}

	LPri, err := self.Comparison.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Next == nil {
		return LPri, nil
	}

	RPri, err := self.Next.Do(context)
	if err != nil {
		return nil, err
	}

	switch self.Op {
	case "!=":
		if IsList(LPri) {
			for _, lval := range LPri.([]Void) {
				if IsString(lval) && IsString(RPri) {
					if lval.(string) == RPri.(string) {
						return false, nil
					}
				} else if IsNumeric(lval) && IsNumeric(RPri) {
					if lval.(float64) == RPri.(float64) {
						return false, nil
					}
				} else if IsBool(lval) && IsBool(RPri) {
					if lval.(bool) == RPri.(bool) {
						return false, nil
					}
				} else if IsNil(lval) && IsNil(RPri) {
					return false, nil
				}
			}
			return true, nil
		} else {
			if IsString(LPri) && IsString(RPri) {
				return (LPri.(string) != RPri.(string)), nil
			} else if IsNumeric(LPri) && IsNumeric(RPri) {
				return (LPri.(float64) != RPri.(float64)), nil
			} else if IsBool(LPri) && IsBool(RPri) {
				return (LPri.(bool) != RPri.(bool)), nil
			} else if IsNil(LPri) {
				if IsNil(RPri) {
					return false, nil
				} else {
					return true, nil
				}
			} else if IsNil(RPri) {
				if IsNil(LPri) {
					return false, nil
				} else {
					return true, nil
				}
			} else {
				return nil, errors.New("!= Op can compare with same value type which are string, numeric, bool")
			}
		}
	case "==":
		if IsList(LPri) {
			for _, lval := range LPri.([]Void) {
				if IsString(lval) && IsString(RPri) {
					if lval.(string) == RPri.(string) {
						return true, nil
					}
				} else if IsNumeric(lval) && IsNumeric(RPri) {
					if lval.(float64) == RPri.(float64) {
						return true, nil
					}
				} else if IsBool(lval) && IsBool(RPri) {
					if lval.(bool) == RPri.(bool) {
						return true, nil
					}
				} else if IsNil(lval) && IsNil(RPri) {
					return true, nil
				}
			}
			return false, nil
		} else {
			if IsString(LPri) && IsString(RPri) {
				return (LPri.(string) == RPri.(string)), nil
			} else if IsNumeric(LPri) && IsNumeric(RPri) {
				return (LPri.(float64) == RPri.(float64)), nil
			} else if IsBool(LPri) && IsBool(RPri) {
				return (LPri.(bool) == RPri.(bool)), nil
			} else if IsNil(LPri) {
				if IsNil(RPri) {
					return true, nil
				} else {
					return false, nil
				}
			} else if IsNil(RPri) {
				if IsNil(LPri) {
					return true, nil
				} else {
					return false, nil
				}
			} else {
				return nil, errors.New("== Op can compare with same value type which are string, numeric, bool")
			}
		}
	case "!~":
		if IsList(LPri) {
			for _, lval := range LPri.([]Void) {
				if IsString(lval) && IsString(RPri) {
					if strings.Contains(lval.(string), RPri.(string)) {
						return false, nil
					}
				} else if IsString(lval) && IsRegexp(RPri) {
					if RPri.(*regexp.Regexp).MatchString(lval.(string)) {
						return false, nil
					}
				}
			}
			return true, nil
		} else {
			if IsString(LPri) && IsString(RPri) {
				return !strings.Contains(LPri.(string), RPri.(string)), nil
			} else if IsString(LPri) && IsRegexp(RPri) {
				return !RPri.(*regexp.Regexp).MatchString(LPri.(string)), nil
			} else {
				return nil, errors.New("!~ Op can compare with same value which are string, regexp")
			}
		}
	case "=~":
		if IsList(LPri) {
			for _, lval := range LPri.([]Void) {
				if IsString(lval) && IsString(RPri) {
					if strings.Contains(lval.(string), RPri.(string)) {
						return true, nil
					}
				} else if IsString(lval) && IsRegexp(RPri) {
					if RPri.(*regexp.Regexp).MatchString(lval.(string)) {
						return true, nil
					}
				}
			}
			return false, nil
		} else {
			if IsString(LPri) && IsString(RPri) {
				return strings.Contains(LPri.(string), RPri.(string)), nil
			} else if IsString(LPri) && IsRegexp(RPri) {
				return RPri.(*regexp.Regexp).MatchString(LPri.(string)), nil
			} else {
				return nil, errors.New("=~ Op can compare with same value which are string, regexp")
			}
		}
	default:
		return nil, errors.New("invalid equality op")
	}
}

type Comparison struct {
	Addition *Addition   `@@`
	Op       string      `[@(">" "=" | ">" | "<" "=" | "<")`
	Next     *Comparison ` @@]`
}

func (self *Comparison) ToString() string {
	if self.Addition == nil {
		return ""
	}

	text := self.Addition.ToString()

	if self.Next == nil {
		return text
	}

	text += " " + self.Op
	text += " " + self.Next.ToString()

	return text
}

func (self *Comparison) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Addition == nil {
		return nil, errors.New("Addition is nil, invalid expression")
	}

	LPri, err := self.Addition.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Next == nil {
		return LPri, nil
	}

	RPri, err := self.Next.Do(context)
	if err != nil {
		return nil, err
	}

	switch self.Op {
	case ">":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) > RPri.(float64)), nil
		}

		return nil, errors.New("> Op can compare with numeric value")
	case ">=":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) >= RPri.(float64)), nil
		}

		return nil, errors.New("> Op can compare with numeric value")
	case "<":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) < RPri.(float64)), nil
		}

		return nil, errors.New("> Op can compare with numeric value")
	case "<=":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) <= RPri.(float64)), nil
		}

		return nil, errors.New("> Op can compare with numeric value")
	default:
		return nil, errors.New("invalid comparison op")
	}
}

type Addition struct {
	Multiplication *Multiplication `@@`
	Op             string          `[@("-" | "+")`
	Next           *Addition       ` @@]`
}

func (self *Addition) ToString() string {
	if self.Multiplication == nil {
		return ""
	}

	text := self.Multiplication.ToString()

	if self.Next == nil {
		return text
	}

	text += " " + self.Op
	text += " " + self.Next.ToString()

	return text
}

func (self *Addition) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Multiplication == nil {
		return nil, errors.New("Multiplication is nil, invalid expression")
	}

	LPri, err := self.Multiplication.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Next == nil {
		return LPri, nil
	}

	RPri, err := self.Next.Do(context)
	if err != nil {
		return nil, err
	}

	switch self.Op {
	case "-":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) - RPri.(float64)), nil
		}

		return nil, errors.New("- Op can calculate with numeric type")
	case "+":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) + RPri.(float64)), nil
		} else if IsString(LPri) && IsString(RPri) {
			return (LPri.(string) + RPri.(string)), nil
		} else if IsList(LPri) && IsList(RPri) {
			return append(LPri.([]Void), RPri.([]Void)...), nil
		} else if IsList(LPri) {
			return append(LPri.([]Void), RPri), nil
		} else if IsList(RPri) {
			tmp := []Void{LPri}
			return append(tmp, RPri.([]Void)...), nil
		}

		return nil, errors.New("+ Op can calculate with same value type which are numeric, string, list")
	default:
		return nil, errors.New("invalid addition op")
	}
}

type Multiplication struct {
	Unary *Unary          `@@`
	Op    string          `[@("/" | "*" | "%")`
	Next  *Multiplication ` @@]`
}

func (self *Multiplication) ToString() string {
	if self.Unary == nil {
		return ""
	}

	text := self.Unary.ToString()

	if self.Next == nil {
		return text
	}

	text += " " + self.Op
	text += " " + self.Next.ToString()

	return text
}

func (self *Multiplication) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Unary == nil {
		return nil, errors.New("Unary is nil, invalid expression")
	}

	LPri, err := self.Unary.Do(context)
	if err != nil {
		return nil, err
	}

	if self.Next == nil {
		return LPri, nil
	}

	RPri, err := self.Next.Do(context)
	if err != nil {
		return nil, err
	}

	switch self.Op {
	case "/":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) / RPri.(float64)), nil
		}

		return nil, errors.New("/ Op can calculate with numeric value")
	case "*":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return (LPri.(float64) * RPri.(float64)), nil
		}

		return nil, errors.New("* Op can calculate with numeric value")
	case "%":
		if IsNumeric(LPri) && IsNumeric(RPri) {
			return float64((int(LPri.(float64)) % int(RPri.(float64)))), nil
		}

		return nil, errors.New("% Op can calculate with numeric value")
	default:
		return nil, errors.New("invalid multiplication op")
	}
}

type Unary struct {
	Op        string     `(@("!" | "NOT" | "-" | "+")`
	Unary     *Unary     ` @@)`
	PrimValue *PrimValue `|@@`
}

func (self *Unary) ToString() string {
	if self.Unary == nil {
		if self.PrimValue == nil {
			return ""
		}
		return self.PrimValue.ToString()
	}

	text := self.Op
	switch strings.ToUpper(self.Op) {
	case "NOT":
		text += " "
	}
	text += self.Unary.ToString()

	return text
}

func (self *Unary) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Unary == nil {
		if self.PrimValue == nil {
			return nil, errors.New("invalid unary primary value")
		}
		return self.PrimValue.Do(context)
	}

	unary, err := self.Unary.Do(context)
	if err != nil {
		return nil, err
	}

	switch strings.ToUpper(self.Op) {
	case "!", "NOT":
		if IsBool(unary) {
			return !unary.(bool), nil
		}

		return nil, errors.New("!, NOT Unary Op can calculate with bool value")
	case "-":
		if IsNumeric(unary) {
			return -unary.(float64), nil
		}

		return nil, errors.New("- Unary Op can calculate with numeric value")
	case "+":
		if IsNumeric(unary) {
			return unary.(float64), nil
		}

		return nil, errors.New("+ Unary Op can calculate with numeric value")
	default:
		return nil, errors.New("invalid unary op")
	}
}
