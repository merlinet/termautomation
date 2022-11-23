package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"encoding/base64"
	"github.com/alecthomas/repr"
	"os"
	"path"
	"strings"
)

const GetRcmdStr = "get"

type Get struct {
	Name        string  `@"get"`
	FileName    string  `@STRING`
	LocalPath   *string `[ @STRING ]`
	SessionName string  `@IDENT`
}

func NewGet(text string) (*Get, *errors.Error) {
	target, err := NewStruct(text, &Get{})
	if err != nil {
		return nil, err
	}
	return target.(*Get), nil
}

func (self *Get) ToString() string {
	if self.LocalPath != nil {
		return fmt.Sprintf("%s %s %s %s", self.Name, self.FileName, *self.LocalPath, self.SessionName)
	} else {
		return fmt.Sprintf("%s %s %s", self.Name, self.FileName, self.SessionName)
	}
}

func (self *Get) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Get) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	sessionnode, err := context.GetSessionNode(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	proc := sessionnode.Proc

	isbash, err := context.IsBash(self.SessionName)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	if !isbash {
		return nil, errors.New("Not support node type or it's not a bash prompt").AddMsg(self.ToString())
	}

	filename, err := context.ReplaceVariable(utils.Unquote(self.FileName))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	var localPath string
	if self.LocalPath != nil {
		localPath, err = context.ReplaceVariable(utils.Unquote(*self.LocalPath))
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		localPath = strings.TrimSpace(localPath)
	}

	err = DoGet(filename, localPath, context.RecordCategory, proc)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	return nil, nil
}

func DoGet(getfilename string, localPath string, category []string, proc *proc.PtyProcess) *errors.Error {
	if len(getfilename) == 0 || proc == nil {
		return errors.New("Invalid arguments")
	}

	/* md5sum 문자열 추출
	 */
	cmd := fmt.Sprintf(`md5sum "%s" `, getfilename)
	outputs, exitCode, err := RemoteCommand(cmd, proc)
	if err != nil {
		return err
	}

	if exitCode != 0 {
		if len(outputs) > 0 {
			return errors.New(fmt.Sprintf("%s, exit code is %d", outputs[0], exitCode))
		} else {
			return errors.New(fmt.Sprintf("%s exit code is %d", cmd, exitCode))

		}
	}
	if len(outputs) == 0 {
		return errors.New(fmt.Sprintf("Invalid md5sum %s output", getfilename))
	}
	remotehash := strings.TrimSpace(strings.Split(outputs[0], " ")[0])

	dir := path.Dir(getfilename)
	filename := path.Base(getfilename)

	/* remote 파일 압축
	 */
	gzipFilename := fmt.Sprintf("%s.tar.gz", filename)

	defer func() {
		RemoteCommand(fmt.Sprintf(`rm -f "%s" `, gzipFilename), proc)
	}()

	cmd = fmt.Sprintf(`tar zcf "%s" -C "%s" "%s" `, gzipFilename, dir, filename)
	outputs, exitCode, err = RemoteCommand(cmd, proc)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		if len(outputs) > 0 {
			return errors.New(fmt.Sprintf("%s, exit code is %d", outputs[0], exitCode))
		} else {
			return errors.New(fmt.Sprintf("%s exit code is %d", cmd, exitCode))

		}
	}

	cmd = fmt.Sprintf(`base64 "%s" `, gzipFilename)
	outputs, exitCode, err = RemoteCommand(cmd, proc)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		if len(outputs) > 0 {
			return errors.New(fmt.Sprintf("%s, exit code is %d", outputs[0], exitCode))
		} else {
			return errors.New(fmt.Sprintf("%s exit code is %d", cmd, exitCode))
		}
	}

	dir, err = config.GetContentsRecordDir(category)
	if err != nil {
		return err
	}

	filepath := fmt.Sprintf("%s/%s", dir, filename)
	gzipFilepath := fmt.Sprintf("%s/%s", dir, gzipFilename)

	fp, oserr := os.OpenFile(gzipFilepath, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	for _, line := range outputs {
		decLine, oserr := base64.StdEncoding.DecodeString(line)
		if oserr != nil {
			fp.Close()
			return errors.New(fmt.Sprintf("%s", oserr))
		}
		fp.Write(decLine)
	}
	fp.Close()

	/* 압축 해제
	 */
	_, err = utils.ExecShell(dir, fmt.Sprintf(`tar zxf "%s" `, gzipFilename))
	if err != nil {
		return err
	}
	defer func() {
		utils.ExecShell(dir, fmt.Sprintf(`rm -f "%s" `, gzipFilename))
		if err != nil {
			utils.ExecShell(dir, fmt.Sprintf(`rm -f "%s" `, filepath))
		}
	}()

	/* hash 비교 확인
	 */
	var hash string
	hash, err = utils.GetFileMd5Hash(filepath)
	if err != nil {
		return err
	}

	if remotehash != hash {
		err = errors.New("Hash doesn't mached, Download file corrupted")
		return err
	}

	if len(localPath) > 0 {
		cmd := fmt.Sprintf(`mv "%s" "%s/%s" `, filepath, dir, localPath)
		msg, e := utils.ExecShell(dir, cmd)
		if e != nil {
			if len(msg) > 0 {
				err = errors.New(fmt.Sprintf("%s, %s, %s", cmd, msg[0], e.Msg))
			} else {
				err = errors.New(fmt.Sprintf("%s, %s", cmd, e.Msg))
			}
			return err
		}
	}

	err = nil
	return err
}

func (self *Get) GetName() string {
	return self.Name
}

func (self *Get) Dump() {
	repr.Println(self)
}
