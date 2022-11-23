package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"strings"

	"github.com/alecthomas/repr"
	"github.com/go-ini/ini"
)

/* ini type
 */
type IniType struct {
	TypeName   string  `@"ini"`
	FilePath   string  `@STRING`
	NameOption *string `[ { @("both_variable_name"|"ignore_section_name") }`
	CompatFlag bool    `  [@"compat_ini"] ]`
}

func (self *IniType) ToString() string {
	text := fmt.Sprintf("%s %s", self.TypeName, self.FilePath)

	if self.NameOption != nil && len(*self.NameOption) > 0 {
		text += fmt.Sprintf(" %s", *self.NameOption)
	}

	if self.CompatFlag {
		text += " compat_ini"
	}

	return text
}

func (self *IniType) Load(context *ReplayerContext) *errors.Error {
	if context == nil {
		return errors.New("invalid arguments")
	}

	if strings.ToLower(self.TypeName) != "ini" {
		return errors.New("invalid load file type, only can ini file.")
	}

	loadfile, err := context.ReplaceVariable(utils.Unquote(self.FilePath))
	if err != nil {
		return err
	}

	path, err := config.GetLoadPath(loadfile, context.RecordCategory)
	if err != nil {
		return err
	}

	conf, goerr := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, path)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	sectionsName := conf.SectionStrings()
	for _, secname := range sectionsName {
		keys := conf.Section(secname).KeyStrings()
		for _, name := range keys {
			value := conf.Section(secname).Key(name).String()

			/* soju에서 정의한 ini 로딩시 ${} 변수를 값으로 변환하여 변수 생성
			 */
			if self.CompatFlag {
				resValue, err := context.ReplaceIniVariable(string(value))
				if err != nil {
					return err
				}
				value = resValue
			}

			if self.NameOption != nil {
				switch strings.ToLower(*self.NameOption) {
				case "both_variable_name":
					err := context.SetVariable(name, string(value), loadfile)
					if err != nil {
						return err
					}

					if strings.ToUpper(secname) != "DEFAULT" {
						/* default가 아닌경우, with section name
						 */
						name2 := fmt.Sprintf("%s:%s", secname, name)

						err := context.SetVariable(name2, string(value), loadfile)
						if err != nil {
							return err
						}
					}
				case "ignore_section_name":
					err := context.SetVariable(name, string(value), loadfile)
					if err != nil {
						return err
					}
				default:
					return errors.New("invalid load name option")
				}
			} else {
				if strings.ToUpper(secname) != "DEFAULT" {
					name = fmt.Sprintf("%s:%s", secname, name)
				}

				err := context.SetVariable(name, string(value), loadfile)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (self *IniType) Unload(context *ReplayerContext) *errors.Error {
	if context == nil {
		return errors.New("invalid arguments")
	}

	if strings.ToLower(self.TypeName) != "ini" {
		return errors.New("invalid load file type, only can ini file.")
	}

	loadfile, err := context.ReplaceVariable(utils.Unquote(self.FilePath))
	if err != nil {
		return err
	}

	return context.DelVariableWithLoadPath(loadfile)
}

func (self *IniType) Dump() {
	repr.Println(self)
}

/* Load 정의
 */
const LoadRcmdStr = "load"

type Load struct {
	Name    string   `@"load"`
	IniType *IniType `@@`
}

func NewLoad(text string) (*Load, *errors.Error) {
	target, err := NewStruct(text, &Load{})
	if err != nil {
		return nil, err
	}
	return target.(*Load), nil
}

func (self *Load) ToString() string {
	str := self.Name

	if self.IniType != nil {
		str += " " + self.IniType.ToString()
	}

	return str
}

func (self *Load) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Load) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.IniType != nil {
		err := self.IniType.Load(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		return nil, nil
	} else {
		return nil, errors.New("invalid load arguments").AddMsg(self.ToString())
	}
}

func (self *Load) GetName() string {
	return self.Name
}

func (self *Load) Dump() {
	repr.Println(self)
}

/* Unload 정의
 */
const UnloadRcmdStr = "unload"

type Unload struct {
	Name    string   `@"unload"`
	IniType *IniType `@@`
}

func NewUnload(text string) (*Unload, *errors.Error) {
	target, err := NewStruct(text, &Unload{})
	if err != nil {
		return nil, err
	}

	return target.(*Unload), nil
}

func (self *Unload) ToString() string {
	if self.IniType != nil {
		return fmt.Sprintf("%s %s", self.Name, self.IniType.ToString())
	}
	return ""
}

func (self *Unload) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Unload) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.IniType != nil {
		err := self.IniType.Unload(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	} else {
		return nil, errors.New("invalid unload arguments").AddMsg(self.ToString())
	}
	return nil, nil
}

func (self *Unload) GetName() string {
	return self.Name
}

func (self *Unload) Dump() {
	repr.Println(self)
}
