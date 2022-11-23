package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const ExpectRcmdStr = "expect"

type Expect struct {
	Name          string  `@"expect"`
	ExpectReFlag  bool    `[ @"r" ]`
	ExpectStr     string  `@STRING`
	ExpectTimeout float64 `@NUMBER`
	SessionName   string  `@IDENT`
}

func NewExpect(text string) (*Expect, *errors.Error) {
	target, err := NewStruct(text, &Expect{})
	if err != nil {
		return nil, err
	}
	return target.(*Expect), nil
}

func (self *Expect) ToString() string {
	reflag := ""
	if self.ExpectReFlag {
		reflag = "r"
	}

	return fmt.Sprintf("%s %s%s %.1f %s", self.Name, reflag, self.ExpectStr, self.ExpectTimeout, self.SessionName)
}

func (self *Expect) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Expect) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	sessionnode, err := context.GetSessionNode(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	proc := sessionnode.Proc

	expectStr, err := context.ReplaceVariable(utils.Unquote(self.ExpectStr))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	matched, promptStr, outputLines, err := DoExpect(proc, self.ExpectTimeout, self.ExpectReFlag,
		expectStr, context.OutputPrintFlag, constdef.MAX_OUTPUT_LINE_COUNT, context.LastPromptStr, true)

	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* expect 일치 안되었으면 return
	 */
	if matched != true {
		return nil, nil
	}

	/* last output 메시지 저장
	 */
	context.LastOutput = outputLines
	context.ExitCode = -1
	context.LastPromptStr = promptStr /* 마지막 prompt string 저장 */
	context.LastOutputSessionName = self.SessionName

	/* output_string, exit_code 값 갱신
	 */
	err = context.SetOutputStringVariable()
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* log 기록
	 * XXX: output 라인이 길어질 경우 비효율적, 느려짐
	 */
	if sessionnode.LogFlag {
		for _, line := range outputLines {
			if sessionnode.LogPrefix == "-" {
				line = fmt.Sprintf("%10s|%s\n", fmt.Sprintf("SEND_%03d", sessionnode.LogSendCount), line)
			} else if len(sessionnode.LogPrefix) > 0 {
				line = fmt.Sprintf("%10s|%s\n", sessionnode.LogPrefix, line)
			} else {
				line = fmt.Sprintf("%s\n", line)
			}

			err := sessionnode.Logger.Write(line)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
		}
	}

	/* expect string match 되고, expect string이 bash이면 exit 코드 확인
	 * XXX: prompt가 "# "로 끝나면 수행
	 */
	isbash, err3 := context.IsBash(self.SessionName)
	if err3 != nil {
		return nil, err3.AddMsg(self.ToString())
	}

	if isbash {
		outputLines1, err1 := ExecRemoteCommand("echo $?", proc, constdef.MAX_OUTPUT_LINE_COUNT, context.LastPromptStr, true)
		if err1 != nil {
			return nil, err1.AddMsg(self.ToString())
		}

		if len(outputLines1) > 0 {
			s, goerr2 := strconv.ParseInt(strings.TrimSpace(outputLines1[len(outputLines1)-1]), 10, 32)
			if goerr2 == nil {
				context.ExitCode = int32(s)

				/* exit_code 값 갱신
				 */
				err := context.SetOutputStringVariable()
				if err != nil {
					return nil, err.AddMsg(self.ToString())
				}
			} else {
				fmt.Println(outputLines1)
				e := errors.New(fmt.Sprintf("%s", goerr2)).AddMsg(self.ToString())
				fmt.Println("ERR:", e.ToString(constdef.DEBUG))
			}
		}
	}

	return nil, nil
}

func DoExpect(process *proc.PtyProcess, expectTimeout float64, expectReFlag bool, expectStr string,
	outputPrintFlag bool, maxOutputLines uint32, lastPromptStr string, lastPromptFlag bool) (bool, string, []string, *errors.Error) {

	if process == nil {
		return false, "", []string{}, errors.New("Invalid arguments")
	}

	outputLines := []string{}
	matchtable := []*proc.LineMatch{}

	if expectReFlag {
		re1, regexpErr := regexp.Compile(expectStr)
		if regexpErr != nil {
			return false, "", []string{}, errors.New(fmt.Sprintf("%s", regexpErr))
		}
		matchtable = append(matchtable, &proc.LineMatch{
			LineType:  proc.LINE_TYPE_PROMPT,
			MatchType: proc.MATCH_TYPE_RE,
			Re:        re1,
			Str:       "",
		})
	} else {
		matchtable = append(matchtable, &proc.LineMatch{
			LineType:  proc.LINE_TYPE_PROMPT,
			MatchType: proc.MATCH_TYPE_STR_EXACT,
			Re:        nil,
			Str:       expectStr,
		})
	}

	sshAuthRe := regexp.MustCompile(`^.*\s+continue connecting \(yes/no.*\)\?\s*`)
	matchtable = append(matchtable, &proc.LineMatch{
		LineType:  proc.LINE_TYPE_SSHAUTH,
		MatchType: proc.MATCH_TYPE_RE,
		Re:        sshAuthRe,
		Str:       "",
	})

	matchtable = append(matchtable, &proc.LineMatch{
		LineType:  proc.LINE_TYPE_MORE,
		MatchType: proc.MATCH_TYPE_STR_CONTAIN,
		Re:        nil,
		Str:       "--More--",
	})

	timeout := expectTimeout * 1000.0
	for {
		rawMsg, lineType, err := process.Read(matchtable, time.Duration(timeout))
		if err != nil {
			return false, "", []string{}, err
		}

		if outputPrintFlag {
			fmt.Printf("%s", rawMsg)
		}

		switch lineType {
		case proc.LINE_TYPE_SSHAUTH:
			err1 := process.Write("yes" + process.Eol)
			if err1 != nil {
				return false, "", []string{}, err1
			}
			continue
		case proc.LINE_TYPE_MORE:
			err1 := process.Write(" ")
			if err1 != nil {
				return false, "", []string{}, err1
			}
			continue
		case proc.LINE_TYPE_OUTPUT_LINE:
			if maxOutputLines > 0 && (uint32(len(outputLines)) >= maxOutputLines) {
				outputLines = outputLines[1:]
			}
			msg := convCharEncoding(rawMsg, process.CharacterSet)
			outputLines = append(outputLines, msg)
		case proc.LINE_TYPE_PROMPT:
			msg := convCharEncoding(rawMsg, process.CharacterSet)
			ll := outputLines
			// prompt echo line 제거
			if len(outputLines) > 0 {
				ll = outputLines[1:]
			}
			return true, msg, ll, nil
		default:
			return false, "", []string{}, errors.New("invalid line type")
		}

	}
}

func convCharEncoding(inputMsg, charset string) string {
	/* CharacterSet 변환
	 */
	msg := inputMsg
	switch strings.ToLower(charset) {
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

	return msg
}

func (self *Expect) GetName() string {
	return self.Name
}

func (self *Expect) Dump() {
	repr.Println(self)
}

/* remote shell
 */
func ExecRemoteCommand(cmd string, proc *proc.PtyProcess, maxOutputLines uint32, lastPromptStr string, lastPromptFlag bool) ([]string, *errors.Error) {
	if proc == nil || len(cmd) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	err := proc.Write(cmd + proc.Eol)
	if err != nil {
		return nil, err
	}
	time.Sleep(time.Millisecond * constdef.SEND_INTERVAL_MILLISECOND)

	_, _, outputLines, err := DoExpect(proc, 60.0, true, constdef.BASH_PROMPT2_RE_STR, false,
		maxOutputLines, lastPromptStr, lastPromptFlag)

	if err != nil {
		return nil, err
	}

	return outputLines, nil
}

func RemoteCommand(cmd string, proc *proc.PtyProcess) ([]string, int, *errors.Error) {
	outputs, err := ExecRemoteCommand(cmd, proc, 0, "", false)
	if err != nil {
		return nil, -1, err
	}

	exitCodeArr, err := ExecRemoteCommand("echo $?", proc, constdef.MAX_OUTPUT_LINE_COUNT, "", false)
	if err != nil {
		return nil, -1, err
	}

	exitCode, oserr := strconv.ParseInt(exitCodeArr[len(exitCodeArr)-1], 10, 32)
	if err != nil {
		return nil, -1, errors.New(fmt.Sprintf("%s", oserr))
	}

	return outputs, int(exitCode), nil
}
