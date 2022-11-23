package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"os"
	"strings"
	"time"

	"github.com/lunixbochs/vtclean"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

/* recorder filter interface
 */
type FilterInterface interface {
	Do(*RecorderContext, string, uint8) bool
}

/* command control filter
 */
type CmdControl struct {
	CmdControlConf *config.CmdControl
	ControlMode    uint8
}

func NewCmdControl(controlmode uint8) (*CmdControl, *errors.Error) {
	switch controlmode {
	case constdef.CMD_CONTROL_MODE_BLOCK:
	case constdef.CMD_CONTROL_MODE_AUTO:
	default:
		return nil, errors.New("Invalid mode arguments")
	}

	cmdControlConf, err := config.NewCmdControl()
	if err != nil {
		return nil, err
	}

	cmdcontrol := CmdControl{
		CmdControlConf: cmdControlConf,
		ControlMode:    controlmode,
	}

	return &cmdcontrol, nil
}

func (self *CmdControl) Do(context *RecorderContext, msg string, _ uint8) bool {
	if context == nil || len(msg) == 0 {
		return true
	}

	cmd := msg

	if self.ControlMode == constdef.CMD_CONTROL_MODE_BLOCK {
		if self.CmdControlConf.IsBlockCmd(cmd) {
			fmt.Println("* We can't record screen control command. like vi. Sorry.")
			context.PrintLastPrompt()
			return false
		}
	} else if self.ControlMode == constdef.CMD_CONTROL_MODE_AUTO {
		/* record terminal cmd(option)에서 !interact 모드 설정시 먼저 적용
		 */
		if context.Mode == constdef.MODE_EXPECT {
			if context.ModeChange == constdef.MODE_INTERACT {
				context.Mode = constdef.MODE_INTERACT
				err := context.GlobalMode.SetInteractMode()
				if err != nil {
					fmt.Println("ERR:", err.ToString(constdef.DEBUG))
				}
			} else {
				if self.CmdControlConf.IsInteractCmd(cmd) {
					/* tcpdump 등은 interact 모드로 동작, expect 모드 전환은 RecordInput 에서 ctrl + c (0x03) 입력시
					 * 전환됨
					 */
					context.Mode = constdef.MODE_INTERACT
					err := context.GlobalMode.SetInteractMode()
					if err != nil {
						fmt.Println("ERR:", err.ToString(constdef.DEBUG))
					}
				}
			}
		}
	}

	return true
}

/* input filter
 */
type RecordInput struct {
	CmdControl *config.CmdControl
}

func NewRecordInput() (*RecordInput, *errors.Error) {
	cmdcontrol, err := config.NewCmdControl()
	if err != nil {
		return nil, err
	}

	recordinput := RecordInput{
		CmdControl: cmdcontrol,
	}

	return &recordinput, nil
}

func (self *RecordInput) Do(context *RecorderContext, msg string, _ uint8) bool {
	if context == nil {
		return true
	}

	/* enableEnterFlag true 인 경우 enter, space record
	 */
	if context.IgnoreSendRcmdFlag {
		if len(strings.TrimSpace(msg)) == 0 {
			return true
		}

		/* ls, clear 등 무의미한 명령어 필터
		 */
		if self.CmdControl.IsIgnoreStr(strings.TrimSpace(msg)) {
			//fmt.Printf("* INFO: %s, recording ignored\n", msg)
			return true
		}
	}

	if context.GlobalMode.IsInteractMode() {
		/* interact mode 인 경우 sleep record cmd 생성
		 * 같은 record에 2개 이상 동시에 기록할때 중복된 sleep time을
		 * 제거하기 위해 record의 last기록 시간으로 sleep time 계산
		 */
		lastModTime, err := context.Logger.GetModTime()
		if err == nil {
			interval := time.Now().Sub(lastModTime)

			sleepStr := fmt.Sprintf("%s %d", SleepRcmdStr, int(float64(interval)/1000000.0))
			sleep, err := NewSleep(sleepStr)
			if err != nil {
				fmt.Println("ERR:", err.ToString(constdef.DEBUG))
				return false
			}

			context.Logger.Write(sleep.ToString(), 3)
		}
	}

	commentMsg := fmt.Sprintf("* %s 에서 \"%s \" 명령어를 실행한다.", context.SessionName, msg)
	comment, err := NewComment(commentMsg)
	if err != nil {
		fmt.Println("ERR:", err.ToString(constdef.DEBUG))
		return false
	}

	send, err := NewSend2(msg, context.SessionName)
	if err != nil {
		fmt.Println("ERR:", err.ToString(constdef.DEBUG))
		return false
	}

	context.Logger.Write(comment.ToString(), 1)
	context.Logger.Write(send.ToString(), 1)
	context.SendExpectFlag = true

	/* 현재 recorder 가 interact 모드인 경우 ctrl + c 발생시
	 * expect 모드 처리
	 */
	if context.Mode == constdef.MODE_INTERACT {
		if len(msg) > 0 && msg[0] == 0x03 {
			context.Mode = constdef.MODE_EXPECT
			context.GlobalMode.SetExpectMode()
		}
	}

	return true
}

/* recorder output filter
 */
type RecordOutput struct {
	OutputBucket string
	Output       []string
	PromptStr    string
}

func NewRecordOutput() (*RecordOutput, *errors.Error) {
	return &RecordOutput{}, nil
}

func (self *RecordOutput) Do(context *RecorderContext, rawMsg string, ioSelecter uint8) bool {
	if context == nil || context.Proc == nil {
		err := errors.New("invalid arguments")
		fmt.Println("ERR:", err.ToString(constdef.DEBUG))
		return false
	}
	proc := context.Proc

	if context.SendExpectFlag == false {
		return true
	}

	switch ioSelecter {
	case constdef.IO_SELECTER_OUTPUT:
		if len(rawMsg) == 0 {
			break
		}

		msg := rawMsg
		switch proc.CharacterSet {
		case "euckr":
			got, _, oserr := transform.String(korean.EUCKR.NewDecoder(), msg)
			if oserr == nil {
				msg = got
			} else {
				fmt.Println("WARN:", errors.New(fmt.Sprintf("%s", oserr)).ToString(constdef.DEBUG))
			}
		case constdef.DEFAULT_CHARACTER_SET:
		default:
		}

		self.OutputBucket += msg
		outputLines := []string{}
		for _, ll := range strings.Split(self.OutputBucket, "\n") {
			line := vtclean.Clean(ll, false)
			/* output line count가 MAX개 이상 발생시 처음것 삭제 후 추가
			 */
			if uint32(len(outputLines)) >= constdef.MAX_RECORDER_OUTPUT_LINE_COUNT {
				outputLines = outputLines[1:]
			}
			outputLines = append(outputLines, line)
		}

		self.OutputBucket = strings.Join(outputLines, "\n")
		if len(outputLines) >= 2 {
			self.Output = outputLines[1 : len(outputLines)-1]
			self.PromptStr = outputLines[len(outputLines)-1]
		}

	case constdef.IO_SELECTER_TIMEOUT:
		/* timeout 발생시, prompt string expect 로 기록
		 */
		receivedBufferSize, err := proc.GetReceivedBufferSize()
		if err != nil {
			fmt.Println("ERR:", err.ToString(constdef.DEBUG))
			return false
		}

		if receivedBufferSize != 0 {
			break
		}

		ok, promptReStr, _ := context.PromptRe.MatchPrompt(self.PromptStr)
		if !ok {
			break
		}

		// prompt string을 주석으로 할까?
		expectComment, err := NewComment(" " + self.PromptStr)
		if err != nil {
			fmt.Println("ERR:", err.ToString(constdef.DEBUG))
			return false
		}

		reflag := ""
		if context.PromptReFlag {
			reflag = "r"
		}

		expectStr := fmt.Sprintf("%s %s%s %.1f %s", ExpectRcmdStr,
			reflag, utils.Quote(promptReStr), constdef.DEFAULT_EXPECT_TIMEOUT, context.SessionName)
		expect, err := NewExpect(expectStr)
		if err != nil {
			fmt.Println("ERR:", err.ToString(constdef.DEBUG))
			return false
		}

		context.Logger.Write(expectComment.ToString(), 1)
		context.Logger.Write(expect.ToString(), 3)

		/* 초기화
		 */
		context.SendExpectFlag = false
		context.LastPromptStr = self.PromptStr
		context.SetOutputExitcode(self.Output, int(0))

		self.OutputBucket = ""
		self.Output = []string{}
		self.PromptStr = ""
	}

	return true
}

/* terminal cmd filter
 */
const (
	TCMD_PREFIX string = "!"

	TCMD_HELP_STR                 string = "help"
	TCMD_EXPECT_STR               string = "expect"
	TCMD_INTERACT_STR             string = "interact"
	TCMD_MODE_STR                 string = "mode"
	TCMD_LOG_STR                  string = "log"
	TCMD_EXIT_STR                 string = "exit"
	TCMD_REC_UI_STR               string = "rec_ui"
	TCMD_REC_REST_STR             string = "rec_rest"
	TCMD_REQUIRE_STR              string = "require"
	TCMD_PUT_STR                  string = "put"
	TCMD_GET_STR                  string = "get"
	TCMD_LIST_STR                 string = "list"
	TCMD_CHECK_STR                string = "check"
	TCMD_SET_IGNORE_SEND_RCMD_STR string = "set_ignore_send_rcmd"
	TCMD_SET_EOL_STR              string = "set_eol"
	TCMD_COMMENT_STR              string = "comment"

	TCMD_TC_SPEC_STR     string = "="
	TCMD_TC_SCENARIO_STR string = "-"
	TCMD_TC_TITLE_STR    string = "%"
	TCMD_TC_COMMENT_STR  string = "#"
	TCMD_TC_STEP_STR     string = "*"
)

type TermCmd struct {
	TcmdTable map[string]func(*RecorderContext, string) (bool, *errors.Error)
}

func NewTermCmd() (*TermCmd, *errors.Error) {
	termcmd := TermCmd{}

	termcmd.TcmdTable = map[string]func(*RecorderContext, string) (bool, *errors.Error){
		TCMD_HELP_STR:                 termcmd.tcmd_help,
		TCMD_EXPECT_STR:               termcmd.tcmd_expect,
		TCMD_INTERACT_STR:             termcmd.tcmd_interact,
		TCMD_MODE_STR:                 termcmd.tcmd_mode,
		TCMD_LOG_STR:                  termcmd.tcmd_log,
		TCMD_EXIT_STR:                 termcmd.tcmd_exit,
		TCMD_REC_UI_STR:               termcmd.tcmd_rec_ui,
		TCMD_REC_REST_STR:             termcmd.tcmd_rec_rest,
		TCMD_REQUIRE_STR:              termcmd.tcmd_require,
		TCMD_PUT_STR:                  termcmd.tcmd_put,
		TCMD_GET_STR:                  termcmd.tcmd_get,
		TCMD_LIST_STR:                 termcmd.tcmd_list,
		TCMD_CHECK_STR:                termcmd.tcmd_check,
		TCMD_SET_IGNORE_SEND_RCMD_STR: termcmd.tcmd_set_ignore_send_rcmd,
		TCMD_SET_EOL_STR:              termcmd.tcmd_set_eol,
		TCMD_COMMENT_STR:              termcmd.tcmd_comment,
	}

	return &termcmd, nil
}

func (self *TermCmd) tcmd_help(_ *RecorderContext, arg string) (bool, *errors.Error) {
	cmdTable := []struct {
		cmdStr     string
		cmdHelpStr string
	}{
		{TCMD_PREFIX + TCMD_HELP_STR, "help"},
		{TCMD_PREFIX + TCMD_EXPECT_STR, "enter expect mode"},
		{TCMD_PREFIX + TCMD_INTERACT_STR, "enter interact mode"},
		{TCMD_PREFIX + TCMD_MODE_STR, "print current mode"},
		{TCMD_PREFIX + TCMD_LOG_STR, "[log prefix], save output"},
		{TCMD_PREFIX + TCMD_EXIT_STR, "exit"},
		{TCMD_PREFIX + TCMD_REC_UI_STR, "record ui cmd, you can exit with ctrl + c"},
		{TCMD_PREFIX + TCMD_REC_REST_STR, "record rest api cmd, you can exit with ctrl + c"},
		{TCMD_PREFIX + TCMD_REQUIRE_STR, "\"record id\", require record that has dependence"},
		{TCMD_PREFIX + TCMD_PUT_STR, "\"file name\", uploading file"},
		{TCMD_PREFIX + TCMD_GET_STR, "\"file name\", downloading file"},
		{TCMD_PREFIX + TCMD_LIST_STR, "display last output string list"},
		{TCMD_PREFIX + TCMD_CHECK_STR, "check output, exit_code which is true"},
		{TCMD_PREFIX + TCMD_SET_IGNORE_SEND_RCMD_STR, "[on|off], on or off ignore send cmd, default on"},
		{TCMD_PREFIX + TCMD_SET_EOL_STR, "[cr|lf|crlf], set end of line character, default lf"},
		{TCMD_TC_SPEC_STR, "Test case specification"},
		{TCMD_TC_SCENARIO_STR, "Test case scenario"},
		{TCMD_TC_TITLE_STR, "Test case sub title"},
		{TCMD_TC_COMMENT_STR, "Test case comment"},
		{TCMD_TC_STEP_STR, "Test case steps"},
	}

	fmt.Println("* Recording terminal commands")
	for _, item := range cmdTable {
		fmt.Printf("  %s: %s\n", item.cmdStr, item.cmdHelpStr)
	}
	return false, nil
}

func (self *TermCmd) tcmd_expect(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil || context.Proc == nil {
		return false, errors.New("* invalid expect arguments")
	}
	proc := context.Proc

	/* expect 모드 전환은 바로 적용
	 */
	if context.Mode == constdef.MODE_INTERACT {
		context.ModeChange = constdef.MODE_EXPECT
		context.Mode = constdef.MODE_EXPECT
		context.GlobalMode.SetExpectMode()
		proc.Write(proc.Eol)
	}

	self.tcmd_mode(context, "")

	return false, nil
}

func (self *TermCmd) tcmd_interact(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid interact arguments")
	}

	/* interact 모드 전환은 바로 cmdcontrolInteract filter에서 수행
	 */
	if context.Mode == constdef.MODE_EXPECT {
		context.ModeChange = constdef.MODE_INTERACT
	}

	self.tcmd_mode(context, "")

	return false, nil
}

func (self *TermCmd) tcmd_mode(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid mode arguments")
	}

	fmt.Printf("* Global Mode: ")
	if context.GlobalMode.IsExpectMode() {
		fmt.Println("EXPECT")
	} else if context.GlobalMode.IsInteractMode() {
		fmt.Println("INTERACT")
	} else {
		fmt.Println("Unknown")
	}

	fmt.Printf("*        Mode: ")
	switch context.ModeChange {
	case constdef.MODE_EXPECT:
		fmt.Println("EXPECT")
	case constdef.MODE_INTERACT:
		fmt.Println("INTERACT")
	default:
		fmt.Println("Unknown")
	}

	return false, nil
}

func (self *TermCmd) tcmd_log(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid log arguments")
	}

	if context.LogFlag == true {
		context.LogFlag = false

		if len(context.LogPrefix) > 0 {
			fmt.Println("* LOG OFF, prefix:", context.LogPrefix)
		} else {
			fmt.Println("* LOG OFF")
		}
	} else {
		context.LogFlag = true

		if len(arg) > 0 {
			context.LogPrefix = arg
			fmt.Println("* LOG ON,  prefix:", context.LogPrefix)
		} else {
			context.LogPrefix = "-"
			fmt.Println("* LOG ON")
		}
	}

	logflagstr := "off"
	if context.LogFlag {
		logflagstr = "on"
	}

	log, err := NewLog(fmt.Sprintf(`log %s "%s" %s`, logflagstr, context.LogPrefix, context.SessionName))
	if err != nil {
		return false, err
	}
	context.Logger.Write(log.ToString(), 3)

	return false, nil
}

func (self *TermCmd) tcmd_exit(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context != nil && context.Proc != nil {
		fmt.Println("Bye!")
		context.Proc.Stop()
	}

	return false, nil
}

func (self *TermCmd) tcmd_rec_ui(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid rec_ui arguments")
	}

	node, err := context.GetNode()
	if err != nil {
		fmt.Println("* WARN:", err.ToString(false))
		return false, nil
	}

	if node.NodeInfo.CanBash() {
		if strings.HasPrefix(context.LastPromptStr, constdef.DEBUG_MODE_BASH_PROMPT) &&
			strings.HasSuffix(context.LastPromptStr, "# ") {

			context.WebuiRecordFlag = true
		} else {
			fmt.Println("* WARN: Not debug shell")
		}
	} else {
		fmt.Println("* WARN: This node can't have bash shell")
		return false, nil
	}

	return true, nil
}

func (self *TermCmd) tcmd_rec_rest(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid rec_rest arguments")
	}

	node, err := context.GetNode()
	if err != nil {
		fmt.Println("* WARN:", err.ToString(false))
		return false, nil
	}

	if node.NodeInfo.CanBash() {
		if strings.HasPrefix(context.LastPromptStr, constdef.DEBUG_MODE_BASH_PROMPT) &&
			strings.HasSuffix(context.LastPromptStr, "# ") {

			context.RestRecordFlag = true
		} else {
			fmt.Println("* WARN: Not debug shell")
		}
	} else {
		fmt.Println("* WARN: This node can't have bash shell")
		return false, nil
	}

	return true, nil
}

func (self *TermCmd) tcmd_require(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil || len(arg) == 0 {
		return false, errors.New("* invalid require arguments")
	}

	requirerid, err := utils.GetQString(arg)
	if err != nil {
		return false, err
	}

	/* check require record exist
	 */
	name, cate, err := utils.ParseRid(requirerid)
	if err != nil {
		return false, err
	}

	requireRecordPath, err := config.GetContentsRecordPath(name, cate)
	if err != nil {
		return false, err
	}

	if _, goerr := os.Stat(requireRecordPath); goerr != nil {
		return false, errors.New(fmt.Sprintf("* %s, invalid record", arg))
	}

	/* check require record
	 */
	currentName := utils.Rid(context.RecordName, context.RecordCategory)
	requireName := utils.Rid(name, cate)

	if currentName == requireName {
		return false, errors.New("* Cann't require current rid")
	}

	require, err := NewRequire(fmt.Sprintf("%s %s", RequireRcmdStr, utils.Quote(requireName)))
	if err != nil {
		return false, err
	}

	// XXX 확인 필요
	_, err = require.Do(context.ReplayerContext)
	fmt.Println() // newline
	if err != nil {
		return false, nil
	}

	/* require record를 수행하고, 결과가 성공이면 record에 추가
	 */
	context.Logger.Write(require.ToString(), 3)

	return false, nil
}

func (self *TermCmd) tcmd_put(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil || len(arg) == 0 || context.Proc == nil {
		return false, errors.New("* invalid require arguments")
	}
	proc := context.Proc

	filename, err := utils.GetQString(arg)
	if err != nil {
		return false, err
	}

	remotePath := ""
	err = DoPut(filename, remotePath, context.RecordCategory, proc)
	if err != nil {
		return false, err
	}

	comment := fmt.Sprintf("* \"%s\" 파일을 %s 로 업로드 한다.", filename, context.SessionName)
	putComment, err := NewComment(comment)
	if err != nil {
		return false, err
	}

	/* require record를 수행하고, 결과가 성공이면 record에 추가
	 */
	put, err := NewPut(fmt.Sprintf("%s \"%s\" %s", PutRcmdStr, filename, context.SessionName))
	if err != nil {
		return false, err
	}

	context.Logger.Write(putComment.ToString(), 1)
	context.Logger.Write(put.ToString(), 3)

	return false, nil
}

func (self *TermCmd) tcmd_get(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil || len(arg) == 0 || context.Proc == nil {
		return false, errors.New("* invalid require arguments")
	}
	proc := context.Proc

	filename, err := utils.GetQString(arg)
	if err != nil {
		return false, err
	}

	localPath := ""
	err = DoGet(filename, localPath, context.RecordCategory, proc)
	if err != nil {
		return false, err
	}

	comment := fmt.Sprintf("* %s 에서 \"%s\" 파일을 다운로드 한다.", context.SessionName, filename)
	getComment, err := NewComment(comment)
	if err != nil {
		return false, err
	}

	/* require record를 수행하고, 결과가 성공이면 record에 추가
	 */
	get, err := NewGet(fmt.Sprintf("%s \"%s\" %s", GetRcmdStr, filename, context.SessionName))
	if err != nil {
		return false, err
	}

	context.Logger.Write(getComment.ToString(), 1)
	context.Logger.Write(get.ToString(), 3)

	return false, nil
}

func (self *TermCmd) tcmd_list(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid require arguments")
	}

	/* 마지막 output 값 출력
	 */
	fmt.Println("* list of output_string variable")
	for i, line := range context.LastOutput {
		fmt.Printf("[%4d] %s\n", i, line)
	}

	return false, nil
}

func (self *TermCmd) tcmd_check(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid require arguments")
	}

	if len(arg) == 0 {
		fmt.Println("ex) check output_string =~ \"eth0\"")
		return false, errors.New("* invalid require arguments")
	}

	check, err := NewCheck(CheckRcmdStr + " " + arg)
	if err != nil {
		return false, err
	}

	res, err := check.Do(context.ReplayerContext)
	if err != nil {
		return false, err
	}
	fmt.Println()

	switch res.(type) {
	case bool:
		//result := res.(bool)
		//if result {
		context.Logger.Write(check.ToString(), 2)
		//}
	}

	return false, nil
}

func (self *TermCmd) tcmd_set_ignore_send_rcmd(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid set ignore send arguments")
	}

	if len(arg) == 0 {
		if context.IgnoreSendRcmdFlag {
			fmt.Println("* IgnoreSendRcmdFlag: ON")
		} else {
			fmt.Println("* IgnoreSendRcmdFlag: OFF")
		}

		return false, nil
	}

	switch strings.ToLower(arg) {
	case "on":
		context.IgnoreSendRcmdFlag = true
	case "off":
		context.IgnoreSendRcmdFlag = false
	default:
		return false, errors.New("Invalid arguments")
	}

	return false, nil
}

func (self *TermCmd) tcmd_set_eol(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil || context.Proc == nil {
		return false, errors.New("* invalid set ignore send arguments")
	}
	proc := context.Proc

	if len(arg) == 0 {
		switch proc.Eol {
		case "\r":
			fmt.Println("* EOL: CR")
		case "\n":
			fmt.Println("* EOL: LF")
		case "\r\n":
			fmt.Println("* EOL: CRLF")
		default:
			return false, errors.New("Invalid EOL")
		}
		return false, nil
	}

	switch strings.ToLower(arg) {
	case constdef.EOL_CR:
		proc.Eol = "\r"
	case constdef.EOL_LF:
		proc.Eol = "\n"
	case constdef.EOL_CRLF:
		proc.Eol = "\r\n"
	default:
		return false, errors.New("Invalid arguments")
	}

	eol, err := NewEol(fmt.Sprintf("%s %s %s", EolRcmdStr, arg, context.SessionName))
	if err != nil {
		return false, err
	}

	context.Logger.Write(eol.ToString(), 3)
	return false, nil
}

func (self *TermCmd) tcmd_comment(context *RecorderContext, arg string) (bool, *errors.Error) {
	if context == nil {
		return false, errors.New("* invalid comment arguments")
	}

	comment, err := NewComment(arg)
	if err != nil {
		return false, err
	}

	context.Logger.Write(comment.ToString(), 1)
	return false, nil
}

func (self *TermCmd) Do(context *RecorderContext, msg string, _ uint8) bool {
	if context == nil || len(msg) <= 0 {
		return true
	}

	prefix := string(msg[0])
	if prefix == TCMD_PREFIX {
		/* options
		 */
		msgStr := strings.TrimSpace(string(msg[1:]))
		cmdStrArray := strings.SplitN(msgStr, " ", 2)
		if len(cmdStrArray) == 0 || len(cmdStrArray[0]) == 0 {
			return true
		}

		cmdStr := strings.TrimSpace(cmdStrArray[0])
		cmdArg := string("")
		if len(cmdStrArray) == 2 {
			cmdArg = strings.TrimSpace(cmdStrArray[1])
		}

		cmdFunc, ok := self.TcmdTable[strings.ToLower(cmdStr)]
		if ok {
			contFlag, err := cmdFunc(context, cmdArg)
			if err != nil {
				fmt.Println("WARN:", err.ToString(constdef.DEBUG), ", continue")
				//return true
			}

			context.PrintLastPrompt()
			return contFlag
		}

	} else if prefix == TCMD_TC_SPEC_STR ||
		prefix == TCMD_TC_SCENARIO_STR ||
		prefix == TCMD_TC_TITLE_STR ||
		prefix == TCMD_TC_COMMENT_STR ||
		prefix == TCMD_TC_STEP_STR {

		/* comment
		 */
		cmdFunc, ok := self.TcmdTable[TCMD_COMMENT_STR]
		if ok {
			contFlag, err := cmdFunc(context, strings.TrimSpace(msg))
			if err != nil {
				fmt.Println("WARN:", err.ToString(constdef.DEBUG), ", continue")
				//return true
			}

			context.PrintLastPrompt()
			return contFlag
		}
	}

	return true
}

/* keyinput filter
 */
type TermInput struct {
}

func NewTermInput() (*TermInput, *errors.Error) {
	return &TermInput{}, nil
}

func (self *TermInput) Do(context *RecorderContext, rawMsg string, _ uint8) bool {
	if context == nil || context.Proc == nil {
		fmt.Println("proc is nil")
		return false
	}
	proc := context.Proc

	msg := rawMsg
	if len(msg) > 0 {
		switch msg[0] {
		case 0x03: // CTRL + C
		default:
			msg += proc.Eol
		}
	} else {
		msg = proc.Eol
	}

	err := proc.Write(msg)
	if err != nil {
		fmt.Println("WARN:", err.ToString(constdef.DEBUG), ", continue")
	}

	return true
}

/* terminal output filter
 */
type TermOutput struct {
}

func NewTermOutput() (*TermOutput, *errors.Error) {
	return &TermOutput{}, nil
}

func (self *TermOutput) Do(context *RecorderContext, rawMsg string, ioSelecter uint8) bool {
	if context == nil || context.Proc == nil {
		return true
	}
	proc := context.Proc
	msg := rawMsg

	/* CharacterSet 변환
	 */
	switch proc.CharacterSet {
	case "euckr":
		got, _, oserr := transform.String(korean.EUCKR.NewDecoder(), msg)
		if oserr == nil {
			msg = got
		} else {
			fmt.Println("WARN:", errors.New(fmt.Sprintf("%s", oserr)).ToString(constdef.DEBUG))
		}
	case constdef.DEFAULT_CHARACTER_SET:
	default:
	}

	switch ioSelecter {
	case constdef.IO_SELECTER_OUTPUT:
		fmt.Printf("%s", msg)
	}

	return true
}
