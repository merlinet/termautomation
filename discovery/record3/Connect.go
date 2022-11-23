package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"time"
)

const ConnectRcmdStr = "connect"

type Connect struct {
	Name        string `@"connect"`
	NodeName    string `@STRING`
	SessionName string `@IDENT`
}

func NewConnect(text string) (*Connect, *errors.Error) {
	target, err := NewStruct(text, &Connect{})
	if err != nil {
		return nil, err
	}
	return target.(*Connect), nil
}

func NewConnect2(nodename string) (*Connect, *errors.Error) {
	if len(nodename) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	connect := Connect{
		Name:        ConnectRcmdStr,
		NodeName:    utils.Quote(nodename),
		SessionName: config.GetSessionId(nodename),
	}
	return &connect, nil
}

func (self *Connect) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.NodeName, self.SessionName)
}

func (self *Connect) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Connect) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	_, ok := context.SessionMap[self.SessionName]
	if ok {
		return nil, errors.New(fmt.Sprintf("%s session already made", self.SessionName)).AddMsg(self.ToString())
	}

	if context.Env == nil {
		return nil, errors.New("environment configuration is not loaded").AddMsg(self.ToString())
	}

	nodename, err := context.ReplaceVariable(utils.Unquote(self.NodeName))
	if err != nil {
		return nil, err
	}

	node := context.Env.GetNode(nodename)
	if node == nil {
		return nil, errors.New(nodename + " node doesn't exist.").AddMsg(self.ToString())
	}

	/* connect 3회 시도
	 */
	connectTry := 0
	var proc *proc.PtyProcess
	var promptstr string
	for {
		connectTry += 1
		proc, promptstr, err = doConnect(node, context.OutputPrintFlag)
		if err != nil {
			if connectTry >= 3 {
				return nil, err.AddMsg(self.ToString())
			}
			time.Sleep(time.Millisecond * 300)
			continue
		}
		break
	}

	sessionnode, err := NewSessionNode(context.LogDir, self.SessionName, proc, node)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	context.SessionMap[self.SessionName] = sessionnode
	context.LastPromptStr = promptstr

	return nil, nil
}

func (self *Connect) Do2(env *config.Env) (*proc.PtyProcess, string, *errors.Error) {
	if env == nil {
		return nil, "", errors.New("invalid arguments")
	}

	node := env.GetNode(utils.Unquote(self.NodeName))
	if node == nil {
		return nil, "", errors.New(utils.Unquote(self.NodeName) + " node doesn't exist.").AddMsg(self.ToString())
	}

	proc, promptstr, err := doConnect(node, true)
	if err != nil {
		return nil, "", err.AddMsg(self.ToString())
	}

	return proc, promptstr, nil
}

/* return: proc
 *         promptstr
 *         error
 */
func doConnect(node *config.Node, outputPrintFlag bool) (*proc.PtyProcess, string, *errors.Error) {
	if node == nil {
		return nil, "", errors.New("Invalid arguments")
	}
	nodeinfo := node.NodeInfo

	command, err := nodeinfo.GetConnectCommand()
	if err != nil {
		return nil, "", err
	}

	eol, err := nodeinfo.GetString("Eol")
	if err != nil {
		return nil, "", err
	}

	characterSet, err := nodeinfo.GetString("CharacterSet")
	if err != nil {
		return nil, "", err
	}

	proc, err := proc.NewPtyProcess(command, characterSet, eol)
	if err != nil {
		return nil, "", err
	}

	err = proc.Start()
	if err != nil {
		return nil, "", err
	}

	/* 자동 로그인 처리
	 */
	promptstr, err := doAutoLogin(proc, node, outputPrintFlag)
	if err != nil {
		return nil, "", err
	}

	return proc, promptstr, nil
}

func (self *Connect) GetName() string {
	return self.Name
}

func (self *Connect) Dump() {
	repr.Println(self)
}

var AUTO_LOGIN_SESSION_NAME string = "UTM_LOGIN"

/* username, password, debug, debug_password가 설정되어 있는 경우 자동 로그인 수행
 * return: promptstr, error
 */
func doAutoLogin(proc *proc.PtyProcess, node *config.Node, outputPrintFlag bool) (string, *errors.Error) {
	if proc == nil || node == nil {
		return "", errors.New("Invalid arguments")
	}

	/* 자동 로그인 Rcmd list를 구함
	 */
	loginRcmdStr, err := GetLoginRcmdList(node)
	if err != nil {
		return "", err
	}

	if len(loginRcmdStr) == 0 {
		return "", nil
	}

	target, err := NewStruct(loginRcmdStr, &RcmdList{})
	if err != nil {
		return "", err
	}
	loginRcmdList := target.(*RcmdList)
	loginRcmdObjList, err := ConvRcmdList2Obj(loginRcmdList)
	if err != nil {
		return "", err
	}

	/* replayer context 생성
	 */
	rcdresult, err := NewRecordResult(0, "AutoLogin", false, "")
	if err != nil {
		return "", err
	}

	loginContext, err := NewReplayerContext("AutoLogin", []string{}, "AutoLogin", outputPrintFlag, rcdresult, "", false, []string{})
	if err != nil {
		return "", err
	}

	sessionNode, err := NewSessionNode("AutoLogin", AUTO_LOGIN_SESSION_NAME, proc, node)
	if err != nil {
		return "", err
	}
	loginContext.SessionMap[AUTO_LOGIN_SESSION_NAME] = sessionNode

	_, err = PlayRcmdList(loginRcmdObjList, loginContext)
	return loginContext.LastPromptStr, err
}

func GetLoginRcmdList(node *config.Node) (string, *errors.Error) {
	if node == nil {
		return "", errors.New("invalid arguments")
	}
	nodeinfo := node.NodeInfo

	expectStr := fmt.Sprintf("expect r%s %.1f %s\n", utils.Quote(constdef.DEFAULT_PROMPT_RE_STR),
		constdef.LOGIN_EXPECT_TIMEOUT, AUTO_LOGIN_SESSION_NAME)

	var loginRcmdStr string

	err := nodeinfo.GetLoginRcmdList(func(needExpectFlag bool, sendStr string) {
		if needExpectFlag {
			loginRcmdStr += expectStr
		}
		loginRcmdStr += fmt.Sprintf("send %s %s\n", utils.Quote(sendStr), AUTO_LOGIN_SESSION_NAME)
	})

	if err != nil {
		return "", err
	}

	loginRcmdStr += expectStr

	return loginRcmdStr, nil
}
