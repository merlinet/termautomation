package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"strings"
)

const BashsetenvRcmdStr = "bashsetenv"

type Bashsetenv struct {
	Name        string `@"bashsetenv"`
	IniName     string `@STRING`
	SessionName string `@IDENT`
}

func NewBashsetenv(text string) (*Bashsetenv, *errors.Error) {
	target, err := NewStruct(text, &Bashsetenv{})
	if err != nil {
		return nil, err
	}
	return target.(*Bashsetenv), nil
}

func (self *Bashsetenv) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.IniName, self.SessionName)
}

func (self *Bashsetenv) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Bashsetenv) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	/* set env를 할 수 있는 bash 상태 인지 검사
	 */
	isbash, err := context.IsBash(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	if !isbash {
		return nil, errors.New("Not support node type or it's not a bash prompt").AddMsg(self.ToString())
	}

	/* variable 에서 loadpath에 해당하는 variable을 찾음
	 */
	iniVariableList, err := context.GetVariableWithLoadPath(utils.Unquote(self.IniName))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	/* expect 객체 생성
	 */
	expectstr := fmt.Sprintf("%s \"%s\" %.1f %s", ExpectRcmdStr, context.LastPromptStr, constdef.LOGIN_EXPECT_TIMEOUT, self.SessionName)
	expect, err := NewExpect(expectstr)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	sendFlag := false
	envFilePath := fmt.Sprintf("/tmp/discovery.env.%s", strings.ToLower(self.SessionName))

	/* 중복 check map
	 */
	envKeyCheckMap := make(map[string]bool)

	for _, elem := range iniVariableList {
		var bashEnvValue, bashEnvKey string

		/* 변수 이름이 section:key 로 되어 있을때 section 문자열 제거
		 */
		sepIndex := strings.Index(elem.Name, ":")
		if sepIndex >= 0 {
			bashEnvKey = strings.TrimSpace(elem.Name[sepIndex+1:])
		} else {
			bashEnvKey = strings.TrimSpace(elem.Name)
		}

		if len(bashEnvKey) == 0 {
			continue
		}

		/* 중복 check
		 */
		_, ok := envKeyCheckMap[bashEnvKey]
		if ok {
			continue
		}
		envKeyCheckMap[bashEnvKey] = true

		/* 값 안에 ${NAME} 같은 변수 string 이 있는 경우, 찾아서 치환
		 */
		var replaceString string

		switch elem.Value.(type) {
		case string:
			replaceString = elem.Value.(string)
		case float64:
			replaceString = fmt.Sprintf("%d", int(elem.Value.(float64)))
		default:
			/* string, float 가 아닌 타임 continue
			 */
			continue
		}

		/* discovery 변수 문자열 치환
		 * ex, $<varname>
		 */
		bashEnvValue, err = context.ReplaceVariable(replaceString)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		/* bash 형태의 변수 문자열 치환
		 * ex, ${varname}
		 */
		bashEnvValue, err = context.ReplaceIniVariable(bashEnvValue)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		var cmd string
		if sendFlag == false {
			cmd = fmt.Sprintf("echo \"export %s=%s\" > %s", bashEnvKey, bashEnvValue, envFilePath)
		} else {
			cmd = fmt.Sprintf("echo \"export %s=%s\" >> %s", bashEnvKey, bashEnvValue, envFilePath)
		}

		/* cmd를 send
		 */
		sendStr := fmt.Sprintf("%s '%s' %s", SendRcmdStr, cmd, self.SessionName)
		send, err := NewSend(sendStr)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		_, err = send.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		_, err = expect.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		sendFlag = true
	}

	/* send 를 수행한 경우에만 expect 수행
	 */
	if sendFlag {
		/* 환경 변수 파일 로드
		 */
		cmd := fmt.Sprintf("source %s", envFilePath)
		sendStr := fmt.Sprintf("%s '%s' %s", SendRcmdStr, cmd, self.SessionName)
		send, err := NewSend(sendStr)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		_, err = send.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}

		_, err = expect.Do(context)
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
	}

	return nil, nil
}

func (self *Bashsetenv) GetName() string {
	return self.Name
}

func (self *Bashsetenv) Dump() {
	repr.Println(self)
}
