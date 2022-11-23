package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"encoding/json"
	"os"
	"strings"
	"time"
)

var DumpContext bool = false

/* Replay Set result 자료구조
 */
type Result struct {
	Testname  string
	Setname   string
	Settype   string
	Timestamp string
	LogDir    string

	Recordcount  uint32
	Successcount uint32
	Failcount    uint32
	Errorcount   uint32

	RecordResults []*RecordResult

	IncompleteRecordList []string // error, fail 발생항 rid list
}

func NewResult(testname, setname, settype, timestamp, logdir string) (*Result, *errors.Error) {
	if len(testname) == 0 || len(setname) == 0 || len(settype) == 0 || len(logdir) == 0 {
		return nil, errors.New("invalid arguments")
	}

	result := Result{
		Testname:      testname,
		Setname:       setname,
		Settype:       settype,
		Timestamp:     timestamp,
		LogDir:        logdir,
		RecordResults: []*RecordResult{},
	}

	return &result, nil
}

func (self *Result) PrintTitle() {
	fmt.Printf("%sㅁ 테스트 실행 이름: %s%s\n", constdef.ANSI_GREEN, self.Testname, constdef.ANSI_END)
	fmt.Printf("%sㅁ   재생 묶음 이름: %s%s\n", constdef.ANSI_GREEN, self.Setname, constdef.ANSI_END)
	fmt.Printf("%sㅁ        실행 시간: %s%s", constdef.ANSI_GREEN, self.Timestamp, constdef.ANSI_END)
}

func (self *Result) AddRecordResult(recordResult *RecordResult) {
	self.RecordResults = append(self.RecordResults, recordResult)
	// record 화면 출력
	recordResult.PrintTitle()
}

func (self *Result) PrintSummary() {
	fmt.Println("\n")
	fmt.Printf("%sㅁ 총 실행 레코드: %d%s\n", constdef.ANSI_GREEN, self.Recordcount, constdef.ANSI_END)
	fmt.Printf("%sㅁ 성공 스텝 개수: %d%s\n", constdef.ANSI_GREEN, self.Successcount, constdef.ANSI_END)
	fmt.Printf("%sㅁ 실패 스텝 개수: %d%s\n", constdef.ANSI_GREEN, self.Failcount, constdef.ANSI_END)
	fmt.Printf("%sㅁ      에러 개수: %d%s\n", constdef.ANSI_GREEN, self.Errorcount, constdef.ANSI_END)
	fmt.Println()
}

func (self *Result) CountResult() {
	for _, rcdresult := range self.RecordResults {
		self.Recordcount++

		s, f, e := rcdresult.GetResult()
		self.Successcount += s
		self.Failcount += f
		self.Errorcount += e

		if f > 0 || e > 0 {
			self.IncompleteRecordList = append(self.IncompleteRecordList, rcdresult.Name)
		}
	}
}

func (self *Result) WriteJson() *errors.Error {
	type ResultJsonForm struct {
		Testname  string
		Setname   string
		Settype   string
		Timestamp string

		Recordcount  uint32
		Successcount uint32
		Failcount    uint32
		Errorcount   uint32

		RecordResults []*RecordResult
	}

	jsonForm := ResultJsonForm{
		Testname:      self.Testname,
		Setname:       self.Setname,
		Settype:       self.Settype,
		Timestamp:     self.Timestamp,
		Recordcount:   self.Recordcount,
		Successcount:  self.Successcount,
		Failcount:     self.Failcount,
		Errorcount:    self.Errorcount,
		RecordResults: self.RecordResults,
	}

	jsonOutput, goerr := json.MarshalIndent(&jsonForm, "", "  ")
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	logpath := fmt.Sprintf("%s/results.json", self.LogDir)

	utils.MakeParentDir(logpath, false)

	fp, goerr := os.OpenFile(logpath, os.O_CREATE|os.O_WRONLY, 0644)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	fmt.Fprintf(fp, "%s", jsonOutput)
	fp.Close()

	fmt.Printf("\n* Result json path: %s\n\n", logpath)

	return nil
}

func (self *Result) WriteIncompleteRecordSet() *errors.Error {
	if len(self.IncompleteRecordList) == 0 {
		return nil
	}

	path := fmt.Sprintf("%s/incomplete.set", self.LogDir)

	utils.MakeParentDir(path, false)

	fp, goerr := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	fmt.Fprintf(fp, "; %s - 실패, 에러 발생 레코드\n", self.Setname)
	for _, rid := range self.IncompleteRecordList {
		fmt.Fprintf(fp, "%s\n", rid)
	}

	return nil
}

/* record result 자료 구조
 */
type RecordResult struct {
	PrintFlag   bool   `json:"-"`
	DepthIndent string `json:"-"`

	Seq         uint32
	Name        string
	StartTime   time.Time
	RunTime     time.Duration
	ErrorResult *ErrorResult
	Steps       []*Step

	Successcount uint32
	Failcount    uint32
	Errorcount   uint32
}

func NewRecordResult(seq uint32, name string, printflag bool, depthindent string) (*RecordResult, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("invalid arguments")
	}

	rcdresult := RecordResult{
		PrintFlag:   printflag,
		DepthIndent: depthindent,

		Seq:       seq,
		Name:      name,
		StartTime: time.Now(),
		Steps:     []*Step{},
	}

	return &rcdresult, nil
}

func (self *RecordResult) CountResult() {
	if self.ErrorResult != nil {
		self.Errorcount++
	}

	for _, step := range self.Steps {
		s, f, e := step.CountResult()
		self.Successcount += s
		self.Failcount += f
		self.Errorcount += e
	}
}

func (self *RecordResult) GetResult() (uint32, uint32, uint32) {
	return self.Successcount, self.Failcount, self.Errorcount
}

func (self *RecordResult) PrintTitle() {
	if self.PrintFlag {
		// 화면 print
		fmt.Println("\n")
		fmt.Printf("%s%s%05d, 레코드 \"%s\"%s", self.DepthIndent, constdef.ANSI_CYAN, self.Seq, self.Name, constdef.ANSI_END)
	}
}

func (self *RecordResult) PrintSummary() {
	if self.PrintFlag {
		fmt.Println()
		fmt.Printf("%s%s성공 스텝: %d%s\n", self.DepthIndent, constdef.ANSI_CYAN, self.Successcount, constdef.ANSI_END)
		fmt.Printf("%s%s실패 스텝: %d%s\n", self.DepthIndent, constdef.ANSI_CYAN, self.Failcount, constdef.ANSI_END)
		fmt.Printf("%s%s     에러: %d%s\n", self.DepthIndent, constdef.ANSI_CYAN, self.Errorcount, constdef.ANSI_END)
		fmt.Printf("%s%s실행 시간: %dms%s\n", self.DepthIndent, constdef.ANSI_CYAN, self.RunTime, constdef.ANSI_END)
	}
}

func (self *RecordResult) AddStep(step *Step) {
	self.Steps = append(self.Steps, step)

	if self.PrintFlag {
		step.PrintMsg()
	}
}

func (self *RecordResult) GetLastCheckStep() *CheckResult {
	if len(self.Steps) == 0 {
		return nil
	}

	lastStep := self.Steps[len(self.Steps)-1]
	switch lastStep.GetType() {
	case "check":
		chkresult := lastStep.Result.(*CheckResult)
		if chkresult.CheckDone || chkresult.CommentType != "*" {
			return nil
		}
		return chkresult
	default:
		return nil
	}
}

func (self *RecordResult) SetResult(errorresult *ErrorResult) {
	self.RunTime = time.Now().Sub(self.StartTime) / time.Millisecond
	self.ErrorResult = errorresult

	if self.PrintFlag {
		// 화면 print
		if errorresult != nil {
			fmt.Printf("\n%s%s\"%s\" 레코드 재생 중 에러가 발생했습니다.%s\n", self.DepthIndent, constdef.ANSI_YELLOW2, self.Name, constdef.ANSI_END)

			fmt.Printf("%s- Error message:\n", self.DepthIndent)
			arr := strings.Split(errorresult.Error.Msg, "\n")
			for i, msg := range arr {
				fmt.Printf("%s| %s", self.DepthIndent, msg)
				if i < len(arr)-1 {
					fmt.Println()
				}
			}

			if DumpContext {
				fmt.Printf("%s- Context dump message:\n", self.DepthIndent)
				arr = strings.Split(errorresult.ContextDump, "\n")
				for i, msg := range arr {
					fmt.Printf("%s| %s", self.DepthIndent, msg)
					if i < len(arr)-1 {
						fmt.Println()
					}
				}
			}
		}
	}
}

/* record 내 step result
 */
type Step struct {
	Type   string // type string: check, require, checker
	Result Void
}

/* step result,
 * check
 * checker
 * require
 * error
 */
func NewStep(result Void) (*Step, *errors.Error) {
	step := Step{
		Result: result,
	}

	switch result.(type) {
	case *CheckResult:
		step.Type = "check"
	case *RecordResult:
		step.Type = "require"
	case *CheckerResult:
		step.Type = "checker"
	case *ErrorResult:
		step.Type = "error"
	default:
		return nil, errors.New("invalid step type string")
	}

	return &step, nil
}

func (self *Step) CountResult() (uint32, uint32, uint32) {
	var success uint32 = 0
	var fail uint32 = 0
	var err uint32 = 0

	switch self.Result.(type) {
	case *CheckResult:
		res := self.Result.(*CheckResult)

		switch res.ResultCode {
		case constdef.SUCCESS:
			success++
		case constdef.FAIL:
			fail++
		}
	case *RecordResult:
		res := self.Result.(*RecordResult)

		res.CountResult()
		s, f, e := res.GetResult()
		success += s
		fail += f
		err += e
	case *CheckerResult:
		res := self.Result.(*CheckerResult)
		switch res.ResultCode {
		case constdef.SUCCESS:
			success++
		case constdef.FAIL:
			fail++
		}
	case *ErrorResult:
		err++
	}

	return success, fail, err
}

func (self *Step) GetType() string {
	return self.Type
}

func (self *Step) GetCheckResult() *CheckResult {
	return self.Result.(*CheckResult)
}

func (self *Step) GetRequireResult() *RecordResult {
	return self.Result.(*RecordResult)
}

func (self *Step) GetCheckerResult() *CheckerResult {
	return self.Result.(*CheckerResult)
}

func (self *Step) GetErrorResult() *ErrorResult {
	return self.Result.(*ErrorResult)
}

func (self *Step) PrintMsg() {
	switch strings.ToLower(self.GetType()) {
	case "check":
		checkresult := self.GetCheckResult()
		checkresult.PrintMsg()
	case "checker":
		checkerresult := self.GetCheckerResult()
		checkerresult.PrintMsg()
	case "error":
		errorresult := self.GetErrorResult()
		errorresult.PrintMsg()

		/*
			case "require":
				requireresult := self.GetRequireResult()
		*/

	}
}

/* check 결과 저장,
 * Message에는 check 바로 전 step comment 가 추가됨
 */
type CheckResult struct {
	PrintFlag   bool   `json:"-"`
	DepthIndent string `json:"-"`

	CommentType         string // =, -, #, %, *, _
	Comment             string // comment
	CheckDone           bool   // check Rcmd 결과가 채워졌는지 구분
	LastSendSessionName string // debug 정보, 마지막 send한 session name
	LastSend            string // 마지막 send 문자열
	OutputSessionName   string
	OutputString        []string // expect 후 output string
	ExitCode            int32    // exit code 값
	CheckCondition      string   // check condition 문자열
	ResultCode          uint8    // check 결과 값
}

func NewCheckResult(commentType, comment string, printflag bool, depthindent string) (*CheckResult, *errors.Error) {
	switch commentType {
	case "=", "-", "#", "%", "*", "_":
	default:
		return nil, errors.New("invalid comment type, =, -, %, *, _ can place")
	}

	chkResult := CheckResult{
		PrintFlag:   printflag,
		DepthIndent: depthindent,

		CommentType:  commentType,
		Comment:      comment,
		CheckDone:    false,
		OutputString: []string{},
		ExitCode:     -1,
		ResultCode:   constdef.NA,
	}

	return &chkResult, nil
}

func (self *CheckResult) PrintMsg() {
	if self.PrintFlag {
		switch self.CommentType {
		case "=":
			fmt.Println()
			fmt.Printf("%s %s# %s%s", self.DepthIndent, constdef.ANSI_CYAN_BOLD, self.Comment, constdef.ANSI_END)
		case "-":
			fmt.Println()
			fmt.Printf("%s %s## %s%s", self.DepthIndent, constdef.ANSI_YELLOW_BOLD, self.Comment, constdef.ANSI_END)
		case "%":
			fmt.Println()
			fmt.Printf("%s %s### %s%s", self.DepthIndent, constdef.ANSI_GREEN, self.Comment, constdef.ANSI_END)
		case "#":
			fmt.Println()
			fmt.Printf("%s %s  |%s%s", self.DepthIndent, constdef.ANSI_WHITE, self.Comment, constdef.ANSI_END)
		case "*":
			fmt.Println()
			fmt.Printf("%s %s[*] %s%s", self.DepthIndent, constdef.ANSI_YELLOW, self.Comment, constdef.ANSI_END)
		case "_":
			fmt.Println()
		}
	}
}

func (self *CheckResult) SetResult(resultCode uint8) {
	self.CheckDone = true
	self.ResultCode = resultCode

	if self.PrintFlag {
		utils.PrintResult(self.ResultCode)
	}
}

func (self *CheckResult) SetInfo(lastsendsessionname string, lastsend string,
	outputsessionname string, output []string, exitcode int32, checkcondition string) {

	self.LastSendSessionName = lastsendsessionname
	self.LastSend = lastsend
	self.OutputSessionName = outputsessionname
	self.OutputString = output
	self.ExitCode = exitcode
	self.CheckCondition = checkcondition

	if self.PrintFlag {
		fmt.Println("")
		if len(self.LastSend) > 0 {
			fmt.Printf("%s     - last send: %s\n", self.DepthIndent, self.LastSend)
		}

		if len(self.OutputString) > 0 {
			fmt.Printf("%s     - output_string:\n", self.DepthIndent)
			for _, msg := range self.OutputString {
				fmt.Printf("%s     |%s\n", self.DepthIndent, msg)
			}
		}

		if self.ExitCode != -1 {
			fmt.Printf("%s     - exit_code: %d\n", self.DepthIndent, self.ExitCode)
		}

		if len(self.CheckCondition) > 0 {
			fmt.Printf("%s     - check condition: %s", self.DepthIndent, self.CheckCondition)
		}
	}
}

/* checker result
 */
type CheckerResult struct {
	PrintFlag   bool   `json:"-"`
	DepthIndent string `json:"-"`

	CheckerPath  string
	OutputString []string
	ResultCode   uint8
}

func NewCheckerResult(checkerpath string, printflag bool, depthindent string) (*CheckerResult, *errors.Error) {
	if len(checkerpath) == 0 {
		return nil, errors.New("invalid arguments")
	}

	checkerresult := CheckerResult{
		PrintFlag:    printflag,
		DepthIndent:  depthindent,
		CheckerPath:  checkerpath,
		OutputString: []string{},
	}

	return &checkerresult, nil
}

func (self *CheckerResult) PrintMsg() {
	if self.PrintFlag {
		fmt.Println()
		fmt.Printf("%s %s[c] %s%s", self.DepthIndent, constdef.ANSI_YELLOW, self.CheckerPath, constdef.ANSI_END)
	}
}

func (self *CheckerResult) SetResult(output []string, resultcode uint8) {
	self.OutputString = output
	self.ResultCode = resultcode

	if self.PrintFlag {
		utils.PrintResult(resultcode)

		fmt.Println("")
		fmt.Printf("%s     - output:\n", self.DepthIndent)
		for i, msg := range self.OutputString {
			fmt.Printf("%s     |%s", self.DepthIndent, msg)
			if i < len(self.OutputString)-1 {
				fmt.Println()
			}
		}
	}
}

/* error result
 */
type ErrorResult struct {
	PrintFlag   bool   `json:"-"`
	DepthIndent string `json:"-"`

	Error       *errors.Error
	ContextDump string
}

func NewErrorResult(err *errors.Error, context *ReplayerContext, printflag bool, depthindent string) (*ErrorResult, *errors.Error) {
	errorresult := ErrorResult{
		PrintFlag:   printflag,
		DepthIndent: depthindent,
	}

	if err != nil {
		errorresult.Error = err
		if context != nil {
			errorresult.ContextDump = context.DumpToString()
		}
	} else {
		errorresult.Error = nil
	}

	return &errorresult, nil
}

func (self *ErrorResult) PrintMsg() {
	if self.PrintFlag {
		// 화면 print
		if self.Error != nil {
			fmt.Printf("\n%s %s[e] RCMD 재생 중 에러가 발생했습니다.%s\n", self.DepthIndent, constdef.ANSI_YELLOW2, constdef.ANSI_END)

			fmt.Printf("%s     - Error message:\n", self.DepthIndent)
			arr := strings.Split(self.Error.Msg, "\n")
			for i, msg := range arr {
				fmt.Printf("%s     | %s", self.DepthIndent, msg)
				if i < len(arr)-1 {
					fmt.Println()
				}
			}

			if DumpContext {
				fmt.Printf("%s     - Context dump message:\n", self.DepthIndent)
				arr = strings.Split(self.ContextDump, "\n")
				for i, msg := range arr {
					fmt.Printf("%s     | %s", self.DepthIndent, msg)
					if i < len(arr)-1 {
						fmt.Println()
					}
				}
			}
		}
	}
}
