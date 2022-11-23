package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"strings"
)

type BPLogin struct {
	Name     string `@"login"`
	NodeName string `@STRING`
}

func (self *BPLogin) ToString() string {
	return fmt.Sprintf("%s %s", self.Name, self.NodeName)
}

func (self *BPLogin) Do(context *ReplayerContext) *errors.Error {
	if context == nil {
		return errors.New("invalid argument")
	}

	if context.BPClient != nil {
		return errors.New("BPClient already init.")
	}

	nodename, err := context.ReplaceVariable(utils.Unquote(self.NodeName))
	if err != nil {
		return err
	}

	node := context.Env.GetNode(nodename)
	if node == nil {
		return errors.New(fmt.Sprintf("%s is invalid node name", nodename))
	}

	if node.NodeType != "bp" {
		return errors.New(fmt.Sprintf("%s is not a breaking point node", nodename))
	}
	nodeinfo := node.NodeInfo

	apiPath, err := nodeinfo.GetString("RestApiPath")
	if err != nil {
		return err
	}

	jsonpath, err := config.GetLoadPath(apiPath, context.RecordCategory)
	if err != nil {
		return err
	}

	/* rest 정보
	 */
	protocol, err := nodeinfo.GetString("RestProtocol")
	if err != nil {
		return err
	}

	ip, err := nodeinfo.GetString("Ip")
	if err != nil {
		return err
	}

	port, err := nodeinfo.GetInt("RestPort")
	if err != nil {
		return err
	}

	/* bp 접속 정보
	 */
	username, err := nodeinfo.GetString("LoginUsername")
	if err != nil {
		return err
	}

	password, err := nodeinfo.GetString("LoginPassword")
	if err != nil {
		return err
	}

	slot, err := nodeinfo.GetInt("Slot")
	if err != nil {
		return err
	}

	ports, err := nodeinfo.GetArrInt("Ports")
	if err != nil {
		return err
	}

	/* bp client 생성
	 */
	bp, err := NewBPClient(protocol, ip, port, username, password, slot, ports, jsonpath)
	if err != nil {
		return err
	}

	/* login, reserveport 수행
	 */
	err = bp.Login()
	if err != nil {
		return err
	}

	err = bp.Reserveports()
	if err != nil {
		bp.Logout()
		return err
	}

	context.BPClient = bp
	return nil
}

/* 옵션이 있는지 check
 * individual_component -> test 에서 개별 component 활성화 후 테스트 실행
 * export_result -> 결과 파일 export 여부
 */
func isOption(options []string, key string) bool {
	for _, opstr := range options {
		if strings.ToLower(opstr) == strings.ToLower(key) {
			return true
		}
	}
	return false
}

func checkOptions(options []string) *errors.Error {
	for _, opstr := range options {
		switch strings.ToLower(opstr) {
		case "individual_component":
		case "export_result":
		default:
			return errors.New(fmt.Sprintf("'%s' invalid bp option string", opstr))
		}
	}
	return nil
}

type BPRfc2544 struct {
	Name     string   `@"rfc2544"`
	TestName string   `@STRING`
	Options  []string `[ { @IDENT } ]`
}

func (self *BPRfc2544) ToString() string {
	text := fmt.Sprintf("%s %s", self.Name, self.TestName)
	if isOption(self.Options, "export_result") {
		text += " " + "export_result"
	}
	return text
}

func (self *BPRfc2544) Do(context *ReplayerContext) *errors.Error {
	if context == nil {
		return errors.New("invalid arguments")
	}

	if context.BPClient == nil {
		return errors.New("BPClient doesn't init.")
	}
	bp := context.BPClient

	testname, err := context.ReplaceVariable(utils.Unquote(self.TestName))
	if err != nil {
		return err
	}

	err = checkOptions(self.Options)
	if err != nil {
		return err
	}

	output, err := bp.RuntestExportResult(testname, MODEL_TYPE_RFC2544, false,
		isOption(self.Options, "export_result"), context)
	if err != nil {
		return err
	}

	// bp 결과 LastOutput, output_string 변수에 설정
	context.LastOutput = output
	context.ExitCode = -1

	err = context.SetOutputStringVariable()
	if err != nil {
		return err
	}

	return nil
}

type BPNormal struct {
	Name     string   `@"normal"`
	TestName string   `@STRING`
	Options  []string `[ { @IDENT } ]`
}

func (self *BPNormal) ToString() string {
	text := fmt.Sprintf("%s %s", self.Name, self.TestName)

	if isOption(self.Options, "individual_component") {
		text += " individual_component"
	}

	if isOption(self.Options, "export_result") {
		text += " export_result"
	}

	return text
}

func (self *BPNormal) Do(context *ReplayerContext) *errors.Error {
	if context == nil {
		return errors.New("invalid arguments")
	}

	if context.BPClient == nil {
		return errors.New("BPClient doesn't init.")
	}
	bp := context.BPClient

	testname, err := context.ReplaceVariable(utils.Unquote(self.TestName))
	if err != nil {
		return err
	}

	err = checkOptions(self.Options)
	if err != nil {
		return err
	}

	output, err := bp.RuntestExportResult(testname, MODEL_TYPE_NORMAL,
		isOption(self.Options, "individual_component"), isOption(self.Options, "export_result"), context)
	if err != nil {
		return err
	}

	// bp 결과 LastOutput, output_string 변수에 설정
	context.LastOutput = output
	context.ExitCode = -1

	err = context.SetOutputStringVariable()
	if err != nil {
		return err
	}

	return nil
}

type BPLogout struct {
	Name string `@"logout"`
}

func (self *BPLogout) ToString() string {
	return fmt.Sprintf("%s", self.Name)
}

func (self *BPLogout) Do(context *ReplayerContext) *errors.Error {
	if context.BPClient == nil {
		return errors.New("BPClient doesn't init.")
	}
	bp := context.BPClient

	err := bp.Unreserveports()
	if err != nil {
		return err
	}

	bp.Logout()
	context.BPClient = nil
	return nil
}

const BPRcmdStr = "bp"

type BP struct {
	Name    string     `@"bp"`
	Login   *BPLogin   `(@@`
	Rfc2544 *BPRfc2544 `|@@`
	Normal  *BPNormal  `|@@`
	Logout  *BPLogout  `|@@)`
}

func NewBP(text string) (*BP, *errors.Error) {
	target, err := NewStruct(text, &BP{})
	if err != nil {
		return nil, err
	}
	return target.(*BP), nil
}

func (self *BP) ToString() string {
	text := self.Name + " "

	if self.Login != nil {
		text += self.Login.ToString()
	} else if self.Rfc2544 != nil {
		text += self.Rfc2544.ToString()
	} else if self.Normal != nil {
		text += self.Normal.ToString()
	} else if self.Logout != nil {
		text += self.Logout.ToString()
	}

	return text
}

func (self *BP) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *BP) Do(context *ReplayerContext) (Void, *errors.Error) {
	if self.Login != nil {
		err := self.Login.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	} else if self.Rfc2544 != nil {
		err := self.Rfc2544.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	} else if self.Normal != nil {
		err := self.Normal.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	} else if self.Logout != nil {
		err := self.Logout.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	} else {
		return nil, errors.New("Invalid BP command").AddMsg(self.ToString())
	}

	return nil, nil
}

func (self *BP) GetName() string {
	return self.Name
}

func (self *BP) Dump() {
	repr.Println(self)
}
