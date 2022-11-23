package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"os"
	"time"
)

/* recorder logger
 */
type RecorderLogger struct {
	LogPath string
	LogFp   *os.File
}

func NewRecorderLogger(cate []string, name string) (*RecorderLogger, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("invalid name")
	}

	logpath, err := config.GetContentsRecordPath(name, cate)
	if err != nil {
		return nil, err
	}

	utils.MakeParentDir(logpath, false)

	fp, goerr := os.OpenFile(logpath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}

	logger := RecorderLogger{
		LogPath: logpath,
		LogFp:   fp,
	}

	return &logger, nil
}

func (self *RecorderLogger) Write(msg string, newLineCount int) *errors.Error {
	if self.LogFp == nil {
		return errors.New("RecorderLogger Fp is null")
	}

	/** file lock 처리는 확인 필요
	utils.SetFileLock(self.LogFp)
	defer utils.SetFileUnlock(self.LogFp)
	*/

	for i := 0; i < newLineCount; i++ {
		msg += "\n"
	}
	_, goerr := self.LogFp.Write([]byte(msg))
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	return nil
}

func (self *RecorderLogger) GetModTime() (time.Time, *errors.Error) {
	fileInfo, goerr := self.LogFp.Stat()
	if goerr != nil {
		return fileInfo.ModTime(), errors.New(fmt.Sprintf("%s", goerr))
	}
	return fileInfo.ModTime(), nil
}

func (self *RecorderLogger) Close() {
	if self.LogFp != nil {
		self.LogFp.Close()
	}
}

/* replayer logger
 */
type ReplayerLogger struct {
	Path string
	Fp   *os.File
}

func NewReplayerLogger(path string) (*ReplayerLogger, *errors.Error) {
	if len(path) == 0 {
		return nil, errors.New("invalid path arguments")
	}

	logger := ReplayerLogger{
		Path: path,
		Fp:   nil,
	}

	return &logger, nil
}

func (self *ReplayerLogger) Write(msg string) *errors.Error {
	if self.Fp == nil {
		utils.MakeParentDir(self.Path, false)

		fp, goerr := os.OpenFile(self.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if goerr != nil {
			return errors.New(fmt.Sprintf("%s", goerr))
		}

		self.Fp = fp
	}

	_, goerr := self.Fp.WriteString(msg)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	return nil
}

func (self *ReplayerLogger) Close() {
	if self.Fp != nil {
		self.Fp.Close()
	}
}
