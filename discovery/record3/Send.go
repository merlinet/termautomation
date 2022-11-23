package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"time"
)

const SendRcmdStr = "send"

type Send struct {
	Name        string `@"send"`
	Command     string `@STRING`
	SessionName string `@IDENT`
}

func NewSend(text string) (*Send, *errors.Error) {
	target, err := NewStruct(text, &Send{})
	if err != nil {
		return nil, err
	}
	return target.(*Send), nil
}

func NewSend2(command string, sessionname string) (*Send, *errors.Error) {
	if len(sessionname) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	send := Send{
		Name:        SendRcmdStr,
		Command:     utils.Quote(command),
		SessionName: sessionname,
	}

	return &send, nil
}

func (self *Send) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.Command, self.SessionName)
}

func (self *Send) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Send) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	sessionnode, err := context.GetSessionNode(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	proc := sessionnode.Proc

	if sessionnode.LogFlag && sessionnode.LogPrefix == "-" {
		sessionnode.LogSendCount += 1
	}

	/* send, send record인 경우 interval 필요
	 */
	defer func() {
		time.Sleep(time.Millisecond * constdef.SEND_INTERVAL_MILLISECOND)
	}()

	command, err := context.ReplaceVariable(utils.Unquote(self.Command))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	if command == "^C" {
		command = string(0x03)
	}

	/* Ctrl + c 는 LastSend 에 저장하지 않음
	 */
	if len(command) > 0 {
		if command[0] == 0x03 {
			return nil, proc.Write(command)
		}

		/* 명령어 send하기전 초기화
		 * Expect에서 값 채움
		 */
		context.LastOutput = []string{}
		context.ExitCode = -1
		context.LastSendSessionName = self.SessionName
		context.LastSend = command
	}

	return nil, proc.Write(command + proc.Eol)
}

func (self *Send) GetName() string {
	return self.Name
}

func (self *Send) Dump() {
	repr.Println(self)
}
