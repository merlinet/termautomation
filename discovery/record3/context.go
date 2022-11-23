package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"gopkg.in/resty.v1"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

/* recorder context
 */
type RecorderContext struct {
	RecordCategory []string
	RecordName     string

	NodeName    string
	SessionName string
	Env         *config.Env
	Proc        *proc.PtyProcess

	Logger *RecorderLogger // recorder logger

	SendExpectFlag bool // Send, Expect pair 맞추기 위한 꼼수

	LastOutput    []string // prompt 이전의 output 메시지
	ExitCode      int      // 현재는 dummy
	LastPromptStr string   // 마지막 prompt string, 꼼수를 위한 변수

	ModeChange uint8
	Mode       uint8
	GlobalMode *Mode

	LogFlag   bool   // !log 명령어 설정 여부
	LogPrefix string // log prefix

	WebuiRecordFlag bool // !rec_ui 명령어 설정 여부
	RestRecordFlag  bool // !rec_rest 명령어 설정 여부

	PromptReFlag bool // prompt match string 이 regex 인지 나타내는 flag
	PromptRe     *config.PromptRegex

	IgnoreSendRcmdFlag bool // enter나 공백, ignore record 명령어 기록 여부

	ReplayerContext *ReplayerContext // recorder 수행중 Rcmd 수행
}

func NewRecorderContext(rid, nodeName, sessionName string, env *config.Env, proc *proc.PtyProcess) (*RecorderContext, *errors.Error) {
	if len(rid) == 0 || len(sessionName) == 0 || env == nil || proc == nil {
		return nil, errors.New("Invalid arguments")
	}

	name, cate, err := utils.ParseRid(rid)
	if err != nil {
		return nil, err
	}

	logger, err := NewRecorderLogger(cate, name)
	if err != nil {
		return nil, err
	}

	/* prompt regex 스트링 얻음
	 */
	promptre, err := config.NewPromptRegex()
	if err != nil {
		return nil, err
	}
	promptReFlag := true

	mode, err1 := NewMode(name, cate)
	if err1 != nil {
		return nil, err1
	}

	context := RecorderContext{
		RecordCategory: cate,
		RecordName:     name,

		NodeName:    nodeName,
		SessionName: sessionName,
		Env:         env,
		Proc:        proc,
		Logger:      logger,

		SendExpectFlag: false,

		ModeChange: constdef.MODE_EXPECT,
		Mode:       constdef.MODE_EXPECT,
		GlobalMode: mode,

		LogFlag: false,

		WebuiRecordFlag: false,

		PromptReFlag: promptReFlag,
		PromptRe:     promptre,

		IgnoreSendRcmdFlag: true,
	}

	err = context.SetupReplayerContext()
	if err != nil {
		return nil, err
	}

	return &context, nil
}

func (self *RecorderContext) SetupReplayerContext() *errors.Error {
	seq := uint32(0)

	rcdresult, err := NewRecordResult(seq, utils.Rid(self.RecordName, self.RecordCategory), true, "")
	if err != nil {
		return err
	}

	dir, err := config.GetContentsResultsDir()
	if err != nil {
		return err
	}

	t := time.Now()
	timestamp := t.Format("20060102150405")
	logdir := fmt.Sprintf("%s/%s/%d", dir, timestamp, seq)

	replayercontext, err := NewReplayerContext(self.RecordName, self.RecordCategory, logdir, false, rcdresult, "", false, []string{})
	if err != nil {
		return err
	}
	self.ReplayerContext = replayercontext

	sessionnode, err := NewSessionNode(logdir, self.SessionName, self.Proc, self.Env.GetNode(self.NodeName))
	if err != nil {
		return err
	}
	self.ReplayerContext.SessionMap[self.SessionName] = sessionnode

	// replayer context에 recorder에서 사용한다는 내부 flag 설정
	self.ReplayerContext.SetRecorderFlag()

	return nil
}

func (self *RecorderContext) SetOutputExitcode(output []string, exitcode int) *errors.Error {
	self.LastOutput = output[:]
	self.ExitCode = exitcode

	varmap := self.ReplayerContext.GetCurrentVarMapSlice()
	if varmap == nil {
		return errors.New("varmap is nil")
	}

	outputVari, err := NewVariable(constdef.OUTPUT_STRING_VARIABLE_NAME, self.LastOutput, "")
	if err != nil {
		return err
	}

	exitcodeVari, err := NewVariable(constdef.EXIT_CODE_VARIABLE_NAME, float64(self.ExitCode), "")
	if err != nil {
		return err
	}

	err = varmap.SetValue(outputVari)
	if err != nil {
		return err
	}

	err = varmap.SetValue(exitcodeVari)
	if err != nil {
		return err
	}

	return nil
}

func (self *RecorderContext) GetNode() (*config.Node, *errors.Error) {
	if len(self.NodeName) == 0 {
		return nil, errors.New("Node doesn't exist.")
	}

	/* node name으로 env의 node 정보 찾음
	 */
	node := self.Env.GetNode(self.NodeName)
	if node == nil {
		return nil, errors.New(self.NodeName + " node doesn't exist.")
	}

	return node, nil
}

/* 꼼수, terminal input filter 가 중단되었을 경우
 * 보기 좋게 하기 위해 출력해줌, 단지 그 목적
 */
func (self *RecorderContext) PrintLastPrompt() {
	if len(self.LastPromptStr) > 0 {
		fmt.Printf("%s", self.LastPromptStr)
	}
}

func (self *RecorderContext) Close() {
	if self.Logger != nil {
		self.Logger.Close()
	}
}

/* replayer context
 */
type ReplayerContext struct {
	RecordCategory []string
	RecordName     string
	LogDir         string

	RecordVersion uint32 /* currently do nothing */

	Args           []string    // replayer arguments
	ForceEnvId     string      // overwrite environment
	NoEnvHashCheck bool        // true인 경우 env hash check를 하지 않음
	Env            *config.Env // environment

	SessionMap map[string]*SessionNode // connect session map

	/* 마지막 Send Rcmd 에서 수행된 문자열
	 */
	LastSendSessionName string
	LastSend            string

	/* 마지막 Expect Rcmd에서 축출된 output 문자열
	 */
	LastOutputSessionName string
	LastOutput            []string

	ExitCode int32 // bash 인 경우 exit code 기록, echo $? 정보

	LastPromptStr string // 마지막 prompt string

	OutputPrintFlag bool // ouput을 replay 화면에 출력할지 여부

	/* string 에서 변수로 치환할 regex
	 */
	VarRe *regexp.Regexp

	/* 변수 관리 key-value map은 replayer context 생성시 0 slice로 생성
	 * table, for 과 같이 nested 구조에서 key-value map을 생성하고 slice에 push 한다
	 * nested 가 끝나면 key-value map을 pop하여 local 변수 처럼 처리 한다
	 * replace variable 수행시 slice max -> 0 으로 탐색하여 nested local 변수부터 search 시작
	 */
	VarMapSlice []VariableMap

	/* internal function map
	 */
	FunctionMap map[string]FunctionInterface

	/* defer rcmd list, defer를 정의할 경우 rcmd 수행시 성공, 실패일 경우 모두 마지막으로 수행됨
	 */
	DeferList []*Defer

	/* record 수행 결과
	 */
	RecordResult *RecordResult

	/* record의 rcmd 수행시 fail 이 나더라도 계속 실행하는 옵션, DEFAULT true
	 */
	FailedButContinue bool

	/* Recorder 에서 사용 여부
	 */
	RecorderFlag bool

	/* breaking poing client 세션
	 * single session
	 */
	BPClient *BPClient

	/* Rest client
	 */
	RestClientMap map[string]*RestClient
}

func NewReplayerContext(recordname string, category []string, logdir string, outputprintflag bool,
	rcdresult *RecordResult, forceEnvId string, noEnvHashCheck bool, args []string) (*ReplayerContext, *errors.Error) {

	if len(recordname) == 0 || len(logdir) == 0 || rcdresult == nil {
		return nil, errors.New("Invalid arguments")
	}

	/* 변수 치환 regex 컴파일
	 */
	varRe, goerr := regexp.Compile(`\$<([^>]*)>`)
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}

	/* internal function 초기화
	 */
	internalFunctionList, err := NewFunctionList()
	if err != nil {
		return nil, err
	}

	context := ReplayerContext{
		RecordName:     recordname,
		RecordCategory: category,
		LogDir:         logdir,
		SessionMap:     make(map[string]*SessionNode),

		OutputPrintFlag: outputprintflag,

		ExitCode: -1, // 초기값

		VarRe:       varRe,
		VarMapSlice: []VariableMap{},
		FunctionMap: internalFunctionList,

		DeferList: []*Defer{},

		RecordResult: rcdresult,

		FailedButContinue: true,

		ForceEnvId:     forceEnvId,
		NoEnvHashCheck: noEnvHashCheck,
		Args:           args,

		RecorderFlag: false,

		RestClientMap: make(map[string]*RestClient),
	}

	// default variable map 생성
	variableMap := NewVariableMap()
	context.PushVarMapSlice(variableMap)

	/* arguments 변수 설정
	 */
	argsVar, err := NewVariable(constdef.ARGS_VARIABLE_NAME, context.Args, "")
	if err != nil {
		return nil, err
	}

	err = variableMap.SetValue(argsVar)
	if err != nil {
		return nil, err
	}

	return &context, nil
}

func (self *ReplayerContext) SetRecorderFlag() {
	self.RecorderFlag = true
	self.FailedButContinue = true
}

func (self *ReplayerContext) GetSessionNode(sessionname string) (*SessionNode, *errors.Error) {
	sessionnode, ok := self.SessionMap[sessionname]
	if ok {
		return sessionnode, nil
	}

	return nil, errors.New(fmt.Sprintf("%s, Invalid session name", sessionname))
}

/* 입력 문자열에 $<varname> 처럼 변수 string을 해당하는 값 문자열로 치환
 * 변수 치환은 recursive하게 수행, 치환된 문자열에 다시 $<name> 문자열이 존재할 경우 ReplaceVariable 함수 다시 호출
 */
func (self *ReplayerContext) ReplaceVariable(msg string) (string, *errors.Error) {
	return self.doReplaceVariable(self.VarRe, msg)
}

/* soju, gauge conf 설정 bash env 설정 할 수 있도록 문자열 치환
 * ${NETWORK:INFIX_TNS_SERVER} 형식 치환
 */
func (self *ReplayerContext) ReplaceIniVariable(msg string) (string, *errors.Error) {
	/* ${} 에서는 배열형식(${}[])은 없으나 regex group을 맞추기 위해 유지
	 */
	varRe, goerr := regexp.Compile(`\${([^}]*)}`)
	if goerr != nil {
		return msg, errors.New(fmt.Sprintf("%s", goerr))
	}

	return self.doReplaceVariable(varRe, msg)
}

/* restful api json 변수 문자열 치환
 * "payload": {
 *    "username": "{username}",
 *    "password": "{password}"
 * }
 */
func (self *ReplayerContext) ReplaceJsonVariable(msg string) (string, *errors.Error) {
	varRe, goerr := regexp.Compile(`{([A-Za-z0-9:]*)}`)
	if goerr != nil {
		return msg, errors.New(fmt.Sprintf("%s", goerr))
	}

	return self.doReplaceVariable(varRe, msg)
}

/* 변수 문자열 regex matched group 을 순회하며 값으로 치환
 */
func (self *ReplayerContext) doReplaceVariable(varRe *regexp.Regexp, msg string) (string, *errors.Error) {
	if varRe == nil {
		return msg, errors.New("Invalid arguments")
	}

	input := msg
	matchedArr := varRe.FindAllStringSubmatch(input, -1)
	if len(matchedArr) == 0 {
		return input, nil
	}

	/* matched 변수 문자열 array
	 */
	for _, item := range matchedArr {
		if len(item) != 2 {
			return input, errors.New("invalid string replacing format")
		}

		from := item[0]
		text := item[1]

		expr, err := NewStruct(text, &Expression{})
		if err != nil {
			return input, err
		}

		res, err := expr.(*Expression).Do(self)
		if err != nil {
			return input, err
		}

		to := fmt.Sprintf("%v", res)
		if f, ok := res.(float64); ok {
			to = strconv.FormatFloat(f, 'f', -1, 64)
		}

		input = strings.Replace(input, from, to, -1)
	}

	return self.doReplaceVariable(varRe, input)
}

/* varmap에서 key 해당하는 variable value 찾음
 */
func (self *ReplayerContext) GetVariable(key string) (*Variable, *errors.Error) {
	if len(key) == 0 {
		return nil, errors.New("invalid arguments")
	}

	for sliceIndex := len(self.VarMapSlice) - 1; sliceIndex >= 0; sliceIndex-- {
		varMap := self.VarMapSlice[sliceIndex]
		value, err := varMap.FindValue(key)
		if err != nil {
			continue
		}

		return value, nil
	}

	return nil, errors.New(fmt.Sprintf("'%v' is invalid variable name", key))
}

func (self *ReplayerContext) GetVariableValue(ident string) (Void, *errors.Error) {
	vari, err := self.GetVariable(ident)
	if err != nil {
		return nil, err
	}

	return vari.Value, nil
}

/* key-value map slice에서 variable의 loadpath 에 해당하는 variable list로 리턴
 */
func (self *ReplayerContext) GetVariableWithLoadPath(ininame string) ([]*Variable, *errors.Error) {
	if len(ininame) == 0 {
		return []*Variable{}, errors.New("invalid arguments")
	}

	list := []*Variable{}

	for sliceIndex := len(self.VarMapSlice) - 1; sliceIndex >= 0; sliceIndex-- {
		varMap := self.VarMapSlice[sliceIndex]
		list = append(list, varMap.FindValueWithLoadPath(ininame)...)
	}

	return list, nil
}

/* var map stack에서 마지막 인자 찾음
 */
func (self *ReplayerContext) GetCurrentVarMapSlice() VariableMap {
	if len(self.VarMapSlice) == 0 {
		return nil
	}
	return self.VarMapSlice[len(self.VarMapSlice)-1]
}

/* varmap slice에 push
 */
func (self *ReplayerContext) PushVarMapSlice(varmap VariableMap) {
	if varmap != nil {
		self.VarMapSlice = append(self.VarMapSlice, varmap)
	}
}

/* varmap slice에 마지막 pop
 */
func (self *ReplayerContext) PopVarMapSlice() {
	if len(self.VarMapSlice) > 0 {
		self.VarMapSlice = self.VarMapSlice[:len(self.VarMapSlice)-1]
	}
}

func (self *ReplayerContext) GetRecordResult() *RecordResult {
	return self.RecordResult
}

/* exit code랑, output_string 변수 생성
 */
func (self *ReplayerContext) SetOutputStringVariable() *errors.Error {
	/* output string을 변수로 사용하기 위해 output string을 var map에 추가
	 */
	currVarMap := self.GetCurrentVarMapSlice()
	if currVarMap == nil {
		return errors.New("Variable map is empty")
	}

	outputStringVar, err := NewVariable(constdef.OUTPUT_STRING_VARIABLE_NAME, self.LastOutput, "")
	if err != nil {
		return err
	}

	err = currVarMap.SetValue(outputStringVar)
	if err != nil {
		return err
	}

	/* exit code 변수 저장
	 */
	exitcodeVari, err := NewVariable(constdef.EXIT_CODE_VARIABLE_NAME, self.ExitCode, "")
	if err != nil {
		return err
	}

	err = currVarMap.SetValue(exitcodeVari)
	if err != nil {
		return err
	}

	return nil
}

/* current var map slice에 변수 값 설정
 */
func (self *ReplayerContext) SetVariable(varname string, value Void, extra string) *errors.Error {
	currVarMap := self.GetCurrentVarMapSlice()
	if currVarMap == nil {
		return errors.New("Variable map is empty")
	}

	variable, err := NewVariable(varname, value, extra)
	if err != nil {
		return err
	}

	return currVarMap.SetValue(variable)
}

/* current var map slice에서 변수 삭제
 */
func (self *ReplayerContext) DelVariable(varName string) *errors.Error {
	varmap := self.GetCurrentVarMapSlice()
	if varmap == nil {
		return errors.New("Variable map is empty")
	}
	return varmap.DelValue(varName)
}

/* current var map slice에서 loadfile에 해당하는 변수 삭제
 */
func (self *ReplayerContext) DelVariableWithLoadPath(loadfile string) *errors.Error {
	varmap := self.GetCurrentVarMapSlice()
	if varmap == nil {
		return errors.New("Variable map is empty")
	}
	return varmap.DelValueWithLoadPath(loadfile)
}

/* context string dump
 */
func (self *ReplayerContext) DumpToString() string {
	return repr.String(self, repr.Indent("  "), repr.OmitEmpty(true),
		repr.IgnoreGoStringer(), repr.Hide(&os.File{}, &regexp.Regexp{}, &exec.Cmd{},
			time.Time{}, &RecordResult{}, &Defer{}, &resty.Client{}, &resty.Response{}))
}

/* context내 session close
 */
func (self *ReplayerContext) Close() {
	if len(self.SessionMap) > 0 {
		for key, sessionnode := range self.SessionMap {
			if sessionnode != nil {
				sessionnode.Close()
			}
			delete(self.SessionMap, key)
		}
	}

	/* breaking point logout
	 */
	if self.BPClient != nil {
		self.BPClient.Close()
		self.BPClient = nil
	}

	/* Rest map
	 */
	if len(self.RestClientMap) > 0 {
		for key, _ := range self.RestClientMap {
			delete(self.RestClientMap, key)
		}
	}
}

/* 현재 prompt 가 bash 인지 확인
 */
func (self *ReplayerContext) IsBash(sessioname string) (bool, *errors.Error) {
	sessionnode, err := self.GetSessionNode(sessioname)
	if err != nil {
		return false, err
	}
	node := sessionnode.Node

	if node == nil {
		/* spawn 인경우
		 * prompt string 검사
		 */
		if strings.HasSuffix(self.LastPromptStr, "# ") || strings.HasSuffix(self.LastPromptStr, "$ ") {
			return true, nil
		}
	} else {
		/* node type 이랑 prompt string 검사
		 */
		if node.NodeInfo.CanBash() && (strings.HasSuffix(self.LastPromptStr, "# ") ||
			strings.HasSuffix(self.LastPromptStr, "$ ")) {

			return true, nil
		}
	}

	return false, nil
}

func (self *ReplayerContext) GetResultOptions() (bool, string) {
	if self.RecordResult != nil {
		return self.RecordResult.PrintFlag, self.RecordResult.DepthIndent
	}
	return false, ""
}
