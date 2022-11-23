package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"reflect"
	"strings"
)

/* Rcmd 구조체 interface
 */
type RcmdInterface interface {
	ToString() string
	Prepare(context *ReplayerContext) *errors.Error
	Do(context *ReplayerContext) (Void, *errors.Error) // return: controlflow bit, error pointer
	GetName() string
	Dump()
}

/* record 구조체
 */
type Record struct {
	Name        string
	Category    []string
	RcmdList    *RcmdList
	RcmdObjList []RcmdInterface
}

/* record의 Rcmd parsing 구조체
 */
type RcmdList struct {
	List []*Rcmd `{ @@ }`
}

/* Rcmd 구조체
 * Rcmd는 하나의 요소만 갖음
 */
type Rcmd struct {
	Bashsetenv  *Bashsetenv  `(@@`
	BP          *BP          `|@@`
	Break       *Break       `|@@`
	Check       *Check       `|@@`
	Close       *Close       `|@@`
	Comment     *Comment     `|@@`
	Connect     *Connect     `|@@`
	Continue    *Continue    `|@@`
	Debug       *Debug       `|@@`
	Defer       *Defer       `|@@`
	Environment *Environment `|@@`
	Eol         *Eol         `|@@`
	Error       *Error       `|@@`
	Expect      *Expect      `|@@`
	For         *For         `|@@`
	Get         *Get         `|@@`
	If          *If          `|@@`
	Load        *Load        `|@@`
	Log         *Log         `|@@`
	Put         *Put         `|@@`
	Return      *Return      `|@@`
	Require     *Require     `|@@`
	Script      *Script      `|@@`
	Send        *Send        `|@@`
	Set         *Set         `|@@`
	Seta        *Seta        `|@@`
	Sleep       *Sleep       `|@@`
	Spawn       *Spawn       `|@@`
	Table       *Table       `|@@`
	Unload      *Unload      `|@@`
	Unset       *Unset       `|@@`
	Version     *Version     `|@@)`
}

func NewRecord(name string, category []string) (*Record, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("invalid name argument")
	}

	record := Record{
		Name:     name,
		Category: category,
	}

	text, err := record.load()
	if err != nil {
		e := errors.New(fmt.Sprintf("%s, %s", utils.Rid(name, category), err.ToString(false)))
		return nil, e
	}

	rcmdlist, err := NewStruct(text, &RcmdList{})
	if err != nil {
		return nil, err
	}

	record.RcmdList = rcmdlist.(*RcmdList)
	record.RcmdObjList, err = ConvRcmdList2Obj(record.RcmdList)
	if err != nil {
		return nil, err
	}

	return &record, nil
}

func (self *Record) load() (string, *errors.Error) {
	path, err := config.GetContentsRecordPath(self.Name, self.Category)
	if err != nil {
		return "", err
	}

	data, oserr := ioutil.ReadFile(path)
	if oserr != nil {
		return "", errors.New(fmt.Sprintf("%s", oserr))
	}
	return string(data), nil
}

func (self *Record) Play(context *ReplayerContext) *errors.Error {
	controlflow, err := PlayRcmdList(self.RcmdObjList, context)
	if err != nil {
		return PlayDeferList(context, err)
	}

	switch controlflow.(type) {
	case int:
		switch controlflow.(int) {
		case CF_RETURN, CF_NOP:
		default:
			err = errors.New("break, continue can place after for, table rcmd")
			return PlayDeferList(context, err)
		}
	}

	return PlayDeferList(context, nil)
}

func (self *Record) Checker(context *ReplayerContext) *errors.Error {
	/* log 디렉토리가 없으면 success 리턴
	 */
	if _, goerr := os.Stat(context.LogDir); goerr != nil {
		return nil
	}

	checkerPath, err := config.GetContentsCheckerPath(self.Name, self.Category)
	if err != nil {
		return err
	}

	checkerList, goerr := filepath.Glob(checkerPath + ".*")
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	if len(checkerList) == 0 {
		return errors.New("There are no checker scripts")
	}

	/* result 출력 옵션 처리
	 */
	printflag, depthindent := context.GetResultOptions()

	for _, checker := range checkerList {
		checkerName := fmt.Sprintf("%s/%s", strings.Join(context.RecordCategory, "/"), path.Base(checker))

		/* checker result 생성
		 */
		checkerResult, err1 := NewCheckerResult(checkerName, printflag, depthindent)
		if err1 != nil {
			return err1
		}

		step, err1 := NewStep(checkerResult)
		if err1 != nil {
			return err1
		}
		context.RecordResult.AddStep(step)

		cmd := exec.Command(checker, context.LogDir)
		stdoutStderr, goerr := cmd.CombinedOutput()
		stdoutMsgArr := strings.Split(string(stdoutStderr), "\n")
		if len(strings.TrimSpace(stdoutMsgArr[len(stdoutMsgArr)-1])) == 0 {
			stdoutMsgArr = stdoutMsgArr[:len(stdoutMsgArr)-1] // cut 마지막 \n
		}

		if goerr != nil {
			checkerResult.SetResult(append(stdoutMsgArr, fmt.Sprintf("ERR: %s", goerr)), constdef.FAIL)
			return nil
		}

		checkerResult.SetResult(stdoutMsgArr, constdef.SUCCESS)
	}

	return nil
}

func (self *Record) Spec() *errors.Error {
	return Spec(self.RcmdList)
}

func (self *Record) FindEnvObj() (*Environment, *errors.Error) {
	for _, rcmdObj := range self.RcmdObjList {
		switch rcmdObj.(type) {
		case *Environment:
			err := rcmdObj.(*Environment).Prepare(nil)
			if err != nil {
				return nil, err
			}

			return rcmdObj.(*Environment), nil
		}
	}

	return nil, errors.New("Cann't find Environment rcmd")
}

func (self *Record) FindEnv() (*config.Env, *errors.Error) {
	environment, err := self.FindEnvObj()
	if err != nil {
		return nil, err
	}

	return environment.Env, nil
}

/* record environemnt setting(overwrite)
 */
func (self *Record) SetEnv(env *config.Env) *errors.Error {
	environment, err := self.FindEnvObj()
	if err != nil {
		return err
	}

	fmt.Println(environment.EnvHash)
	fmt.Println(env.EnvHash)

	environment.Env = env
	return nil
}

func (self *Record) Dump() {
	fmt.Println("Name: ", self.Name)
	fmt.Println("Category: ", self.Category)

	for _, rcmdObj := range self.RcmdObjList {
		rcmdObj.Dump()
	}
}

/* RcmdList parsing struct 에서 Rcmd Obj로 변환
 */
func ConvRcmdList2Obj(rcmdlist *RcmdList) ([]RcmdInterface, *errors.Error) {
	rcmdObjList := []RcmdInterface{}

	if rcmdlist == nil {
		return rcmdObjList, nil
	}

	/* Rcmd -> RcmdInterface 얻음
	 * Prepare 실행
	 */
	for _, rcmd := range rcmdlist.List {
		rcmdObj, err := GetRcmdObj(rcmd)
		if err != nil {
			return nil, err
		}

		rcmdObjList = append(rcmdObjList, rcmdObj)
	}

	return rcmdObjList, nil
}

/* Rcmd Obj List 실행
 */
func PlayRcmdList(rcmdObjList []RcmdInterface, context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments")
	}

	/* Prepare 실행
	 */
	for _, rcmdObj := range rcmdObjList {
		err := rcmdObj.Prepare(context)
		if err != nil {
			return nil, err
		}
	}

	/* Do 실행
	 */
	for _, rcmdObj := range rcmdObjList {
		controlflow, err := rcmdObj.Do(context)
		if err != nil {
			return controlflow, err
		}

		/* control flow 가 nil, NOP 이 아니면
		 * return 해서, table, for문에서 check
		 */
		switch controlflow.(type) {
		case int:
			switch controlflow.(int) {
			case CF_BREAK, CF_CONTINUE, CF_RETURN:
				return controlflow, nil
			}
		}
	}

	return nil, nil
}

/* Rcmd struct 는 parsing 조건에 따라 하나의 rcmd 만 nil 이 아님
 * 각 field에서 nil이 아닌 개별 rcmd를 찾아 Rcmd Interface 를 return 함
 */
func GetRcmdObj(rcmd *Rcmd) (RcmdInterface, *errors.Error) {
	if rcmd == nil {
		return nil, errors.New("Invalid arguments")
	}

	rcmdFields := reflect.TypeOf(*rcmd)
	rcmdValues := reflect.ValueOf(*rcmd)
	rcmdFieldAmount := rcmdFields.NumField()

	for idx := 0; idx < rcmdFieldAmount; idx++ {
		rcmdObj, err := FieldValueToRcmdInterface(rcmdValues.Field(idx).Interface())
		if err != nil {
			return nil, err
		}

		if reflect.ValueOf(rcmdObj).IsNil() {
			continue
		}

		return rcmdObj, nil
	}

	return nil, errors.New("Cann't find rcmd field value")
}

func FieldValueToRcmdInterface(fieldValue interface{}) (RcmdInterface, *errors.Error) {
	var obj RcmdInterface

	switch fieldValue.(type) {
	case *Bashsetenv:
		obj = fieldValue.(*Bashsetenv)
	case *BP:
		obj = fieldValue.(*BP)
	case *Break:
		obj = fieldValue.(*Break)
	case *Check:
		obj = fieldValue.(*Check)
	case *Close:
		obj = fieldValue.(*Close)
	case *Comment:
		obj = fieldValue.(*Comment)
	case *Connect:
		obj = fieldValue.(*Connect)
	case *Continue:
		obj = fieldValue.(*Continue)
	case *Debug:
		obj = fieldValue.(*Debug)
	case *Defer:
		obj = fieldValue.(*Defer)
	case *Environment:
		obj = fieldValue.(*Environment)
	case *Eol:
		obj = fieldValue.(*Eol)
	case *Error:
		obj = fieldValue.(*Error)
	case *Expect:
		obj = fieldValue.(*Expect)
	case *For:
		obj = fieldValue.(*For)
	case *Get:
		obj = fieldValue.(*Get)
	case *If:
		obj = fieldValue.(*If)
	case *Load:
		obj = fieldValue.(*Load)
	case *Log:
		obj = fieldValue.(*Log)
	case *Put:
		obj = fieldValue.(*Put)
	case *Return:
		obj = fieldValue.(*Return)
	case *Require:
		obj = fieldValue.(*Require)
	case *Script:
		obj = fieldValue.(*Script)
	case *Send:
		obj = fieldValue.(*Send)
	case *Set:
		obj = fieldValue.(*Set)
	case *Seta:
		obj = fieldValue.(*Seta)
	case *Sleep:
		obj = fieldValue.(*Sleep)
	case *Spawn:
		obj = fieldValue.(*Spawn)
	case *Table:
		obj = fieldValue.(*Table)
	case *Unload:
		obj = fieldValue.(*Unload)
	case *Unset:
		obj = fieldValue.(*Unset)
	case *Version:
		obj = fieldValue.(*Version)
	default:
		return nil, errors.New(fmt.Sprintf("%s, invalid rcmd", reflect.TypeOf(fieldValue).String()))
	}

	return obj, nil
}

func PlayDeferList(context *ReplayerContext, prevErr *errors.Error) *errors.Error {
	if len(context.DeferList) == 0 {
		return prevErr
	}

	if prevErr != nil {
		/* defer 수행 이전 error가 있으면
		 * record result 에 error step 으로 추가하고 실행
		 */
		printflag, depthindent := context.GetResultOptions()
		errorsresult, err := NewErrorResult(prevErr, context, printflag, depthindent)
		if err != nil {
			return err
		}

		step, err := NewStep(errorsresult)
		if err != nil {
			return err
		}
		context.RecordResult.AddStep(step)
	}

	/* defer rcmd list 수행
	 */
	for _, rcmdobj := range context.DeferList {
		controlflow, err := rcmdobj.Do2(context)
		if err != nil {
			return err
		}

		switch controlflow.(type) {
		case int:
			switch controlflow.(int) {
			case CF_BREAK, CF_CONTINUE:
				return errors.New("break, continue can place after for, table rcmd")
			case CF_RETURN:
				break
			}
		}
	}

	return nil
}

func Spec(rcmdlist *RcmdList) *errors.Error {
	rcmdObjList, err := ConvRcmdList2Obj(rcmdlist)
	if err != nil {
		return err
	}

	for _, rcmdObj := range rcmdObjList {
		err := SpecRcmd(rcmdObj)
		if err != nil {
			return err
		}
	}

	return nil
}

func SpecRcmd(rcmd RcmdInterface) *errors.Error {
	if rcmd == nil {
		return errors.New("Invalid arguments")
	}

	switch rcmd.(type) {
	case *Table:
		err := Spec(rcmd.(*Table).RcmdList)
		if err != nil {
			return err
		}
	case *For:
		err := Spec(rcmd.(*For).RcmdList)
		if err != nil {
			return err
		}
	case *Comment:
		commentRcmd := rcmd.(*Comment)
		comment := strings.TrimSpace(commentRcmd.Name[1:])
		if len(comment) <= 1 {
			break
		}

		specDepth := comment[0]
		specComment := strings.TrimSpace(comment[1:])
		switch specDepth {
		case '=':
			fmt.Printf("%s\n", specComment)
			fmt.Printf("===================================\n\n")
		case '-':
			fmt.Printf("\n%s\n", specComment)
			fmt.Printf("-----------------------------------\n\n")
		case '%':
			fmt.Printf("### %s\n", specComment)
		case '#':
			fmt.Printf("    %s\n", specComment)
		case '*':
			fmt.Printf(" * %s\n", specComment)
		}
	}

	return nil
}
