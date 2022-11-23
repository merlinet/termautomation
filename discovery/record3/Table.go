package record3

import (
	"bufio"
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"encoding/csv"
	"github.com/alecthomas/repr"
	"io"
	"os"
	"strings"
)

const TableRcmdStr = "table"

type Csv struct {
	FileType string `@"csv"`
	FilePath string `@STRING`
}

func (self *Csv) ToString() string {
	return fmt.Sprintf("%s %s", self.FileType, self.FilePath)
}

func (self *Csv) Do(rcmdObjList []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments")
	}

	filepath, err := context.ReplaceVariable(utils.Unquote(self.FilePath))
	if err != nil {
		return nil, err
	}

	loadpath, err := config.GetLoadPath(filepath, context.RecordCategory)
	if err != nil {
		return nil, err
	}

	fp, goerr := os.Open(loadpath)
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	csvReader := csv.NewReader(bufio.NewReader(fp))

	nameArr, goerr := csvReader.Read()
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}

	if len(nameArr) == 0 {
		return nil, errors.New("Invalid csv data")
	}

	varmap := NewVariableMap()
	context.PushVarMapSlice(varmap)
	defer context.PopVarMapSlice()

	for {
		valueArr, goerr := csvReader.Read()
		if goerr != nil {
			if goerr == io.EOF {
				break
			}
			return nil, errors.New(fmt.Sprintf("%s", goerr))
		}

		err1 := varmap.SetTable(nameArr, valueArr, utils.Unquote(self.FilePath))
		if err1 != nil {
			return nil, err1
		}

		controlflow, err := PlayRcmdList(rcmdObjList, context)
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

	return nil, nil
}

type Row struct {
	FileType string `@"row"`
	VarName  string `@IDENT`
	FilePath string `@STRING`
}

func (self *Row) ToString() string {
	return fmt.Sprintf("%s %s %s", self.FileType, self.VarName, self.FilePath)
}

func (self *Row) Do(rcmdObjList []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments")
	}

	filepath, err := context.ReplaceVariable(utils.Unquote(self.FilePath))
	if err != nil {
		return nil, err
	}

	loadpath, err := config.GetLoadPath(filepath, context.RecordCategory)
	if err != nil {
		return nil, err
	}

	fp, goerr := os.Open(loadpath)
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	varmap := NewVariableMap()
	context.PushVarMapSlice(varmap)
	defer context.PopVarMapSlice()

	reader := bufio.NewReader(fp)

	for {
		data, _, goerr := reader.ReadLine()
		if goerr != nil {
			if goerr == io.EOF {
				break
			}
			return nil, errors.New(fmt.Sprintf("%s", goerr))
		}

		line := strings.TrimSpace(string(data))
		if len(line) == 0 {
			continue
		}

		/* XX:load path 설정 안함
		 */
		var1, err := NewVariable(self.VarName, line, "")
		if err != nil {
			return nil, err
		}

		err = varmap.SetValue(var1)
		if err != nil {
			return nil, err
		}

		controlflow, err := PlayRcmdList(rcmdObjList, context)
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

	return nil, nil
}

type Table struct {
	Name            string    `@"table"`
	Csv             *Csv      `(@@`
	Row             *Row      `|@@)`
	RcmdList        *RcmdList `@@`
	EndtableKeyword string    `@"endtable"`

	RcmdObjList []RcmdInterface
}

func NewTable(text string) (*Table, *errors.Error) {
	target, err := NewStruct(text, &Table{})
	if err != nil {
		return nil, err
	}
	return target.(*Table), nil
}

func (self *Table) ToString() string {
	text := self.Name
	if self.Csv != nil {
		text += " " + self.Csv.ToString()
	} else if self.Row != nil {
		text += " " + self.Row.ToString()
	}

	/* XXX
	if self.RcmdList != nil {
		text += "\n"
		text += ToStringRcmdList(self.RcmdList)
	}

	text += self.EndtableKeyword
	*/

	return text
}

func (self *Table) Prepare(context *ReplayerContext) *errors.Error {
	rcmdobjlist, err := ConvRcmdList2Obj(self.RcmdList)
	if err != nil {
		return err
	}

	self.RcmdObjList = rcmdobjlist
	return nil
}

func (self *Table) Do(context *ReplayerContext) (Void, *errors.Error) {
	/* table csv 형태인 경우
	 */
	if self.Csv != nil {
		controlflow, err := self.Csv.Do(self.RcmdObjList, context)
		if err != nil {
			return controlflow, err.AddMsg(self.ToString())
		}
		return controlflow, nil
	} else if self.Row != nil {
		controlflow, err := self.Row.Do(self.RcmdObjList, context)
		if err != nil {
			return controlflow, err.AddMsg(self.ToString())
		}
		return controlflow, nil
	}
	return nil, errors.New("invalid table arguments").AddMsg(self.ToString())
}

func (self *Table) GetName() string {
	return self.Name
}

func (self *Table) Dump() {
	repr.Println(self)
}
