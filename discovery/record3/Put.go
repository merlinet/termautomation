package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"github.com/alecthomas/repr"
	"os"
	"path"
	"strings"
)

const PutRcmdStr = "put"

type Put struct {
	Name        string  `@"put"`
	FileName    string  `@STRING`
	RemotePath  *string `[ @STRING ]`
	SessionName string  `@IDENT`
}

func NewPut(text string) (*Put, *errors.Error) {
	target, err := NewStruct(text, &Put{})
	if err != nil {
		return nil, err
	}
	return target.(*Put), nil
}

func (self *Put) ToString() string {
	if self.RemotePath != nil {
		return fmt.Sprintf("%s %s %s %s", self.Name, self.FileName, *self.RemotePath, self.SessionName)
	} else {
		return fmt.Sprintf("%s %s %s", self.Name, self.FileName, self.SessionName)
	}
}

func (self *Put) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Put) Do(context *ReplayerContext) (Void, *errors.Error) {
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

	var remotePath string
	if self.RemotePath != nil {
		remotePath, err = context.ReplaceVariable(utils.Unquote(*self.RemotePath))
		if err != nil {
			return nil, err.AddMsg(self.ToString())
		}
		remotePath = strings.TrimSpace(remotePath)
	}

	err = DoPut(filename, remotePath, context.RecordCategory, proc)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	return nil, nil
}

func DoPut(putfilepath string, remotePath string, category []string, proc *proc.PtyProcess) *errors.Error {
	if len(putfilepath) == 0 || strings.TrimSpace(putfilepath)[0] == '/' ||
		strings.TrimSpace(putfilepath) == "." || proc == nil {

		return errors.New("Invalid arguments")
	}

	filepath, err := config.GetLoadPath(putfilepath, category)
	if err != nil {
		return err
	}

	st, oserr := os.Stat(filepath)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}
	if st.IsDir() {
		return errors.New(fmt.Sprintf("%s is directory", filepath))
	}

	dir := path.Dir(filepath)
	filename := path.Base(filepath)

	gzipFilename := fmt.Sprintf("%s.tar.gz", filename)
	gzipFilepath := fmt.Sprintf("%s/%s", dir, gzipFilename)

	/* return error 값 초기화 선언
	 */
	var reterror *errors.Error = nil

	defer func() {
		/* local gzip file 삭제
		 */
		utils.ExecShell(dir, fmt.Sprintf(`rm -f "%s" `, gzipFilename))

		/* 원격 파일 처리 에러 발생시 gzipFilename, filename 파일 삭제
		 */
		if reterror != nil {
			cmd := fmt.Sprintf(`rm -rf "%s" "%s.base64" "%s" `, gzipFilename, gzipFilename, filename)
			RemoteCommand(cmd, proc)
		}
	}()

	// 파일 압축
	_, err = utils.ExecShell(dir, fmt.Sprintf(`tar zcf "%s" "%s" `, gzipFilename, filename))
	if err != nil {
		return err
	}

	base64DataArr, err := utils.GetFileBase64(gzipFilepath)
	if err != nil {
		return err
	}

	msgArr := []string{}
	msgArr = append(msgArr, "stty -echo")
	msgArr = append(msgArr, fmt.Sprintf("cat <<__END__ > \"%s.base64\"", gzipFilename))
	err = sendMsgArr(msgArr, proc)
	if err != nil {
		reterror = err
		return reterror
	}

	for _, msg := range base64DataArr {
		err = proc.Write(msg + proc.Eol)
		if err != nil {
			reterror = err
			return reterror
		}

		_, _, _, err := DoExpect(proc, 10.0, true, constdef.BASH_PROMPT_RE_STR, false, constdef.MAX_OUTPUT_LINE_COUNT, "", false)
		if err != nil {
			reterror = err
			return reterror
		}
	}

	msgArr = []string{}
	msgArr = append(msgArr, "__END__")
	msgArr = append(msgArr, fmt.Sprintf(`base64 -d "%s.base64" > "%s" `, gzipFilename, gzipFilename))
	msgArr = append(msgArr, fmt.Sprintf(`tar zxf "%s" `, gzipFilename))
	msgArr = append(msgArr, fmt.Sprintf(`rm -f "%s.base64" `, gzipFilename))
	msgArr = append(msgArr, fmt.Sprintf(`rm -f "%s" `, gzipFilename))
	msgArr = append(msgArr, fmt.Sprintf("chown `id -u`:`id -g` \"%s\" ", filename))
	msgArr = append(msgArr, "stty sane")
	err = sendMsgArr(msgArr, proc)
	if err != nil {
		reterror = err
		return reterror
	}

	/* upload된 파일의 md5sum hash값 비교
	 */
	cmd := fmt.Sprintf(`md5sum "%s" `, filename)
	outputLines, exitCode, err := RemoteCommand(cmd, proc)
	if err != nil {
		reterror = err
		return reterror
	}

	if exitCode != 0 {
		if len(outputLines) > 0 {
			reterror = errors.New(fmt.Sprintf("%s, %s", cmd, outputLines[0]))
		} else {
			reterror = errors.New(fmt.Sprintf("%s, exit code is %d", cmd, exitCode))
		}
		return reterror
	}

	if len(outputLines) == 0 {
		reterror = errors.New(fmt.Sprintf("%s, Invalid result", cmd))
		return reterror
	}

	/* 업로드된 파일의 hash값 비교
	 */
	remoteMd5sum := strings.Split(outputLines[0], " ")[0]

	/* 파일 hash 값 얻기
	 */
	uploadFileHash, err := utils.GetFileMd5Hash(filepath)
	if err != nil {
		reterror = err
		return reterror
	}

	if remoteMd5sum != uploadFileHash {
		reterror = errors.New("uploading file hash mismached with original file's hash")
		return reterror
	}

	/* remotePath 로 이동
	 */
	if len(remotePath) > 0 {
		cmd := fmt.Sprintf("mv \"%s\" \"%s\" ", filename, remotePath)
		outputLines, exitCode, err := RemoteCommand(cmd, proc)
		if err != nil {
			reterror = err
			return reterror
		}

		if exitCode != 0 {
			if len(outputLines) > 0 {
				reterror = errors.New(fmt.Sprintf("%s, %s", cmd, outputLines[0]))
			} else {
				reterror = errors.New(fmt.Sprintf("%s, exit code is %d", cmd, exitCode))
			}
			return reterror
		}
	}

	reterror = nil
	return reterror
}

func sendMsgArr(msgArr []string, proc *proc.PtyProcess) *errors.Error {
	if proc == nil {
		return errors.New("Invalid arguments")
	}

	for _, msg := range msgArr {
		err := proc.Write(msg + proc.Eol)
		if err != nil {
			return err
		}

		_, _, _, err = DoExpect(proc, 60.0, true, constdef.BASH_PROMPT_RE_STR, false, constdef.MAX_OUTPUT_LINE_COUNT, "", false)
		if err != nil {
			return err
		}
	}

	return nil
}

func (self *Put) GetName() string {
	return self.Name
}

func (self *Put) Dump() {
	repr.Println(self)
}
