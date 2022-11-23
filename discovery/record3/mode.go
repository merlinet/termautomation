package record3

import (
	"bufio"
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"os"
	"strconv"
)

/* recorder는 개별 프로세스로 구동 되기 때문에 mode를 공유하기 위해 파일 생성
 */
type Mode struct {
	RecordName     string
	RecordCategory []string

	FilePath string
}

func NewMode(name string, category []string) (*Mode, *errors.Error) {
	if len(name) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	recordDir, err := config.GetContentsRecordDir(category)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/.%s.mode", recordDir, name)

	mode := Mode{
		RecordName:     name,
		RecordCategory: category,
		FilePath:       path,
	}

	return &mode, nil
}

func (self *Mode) getCount() uint32 {
	fp, oserr := os.Open(self.FilePath)
	if oserr != nil {
		return uint32(0)
	}
	defer fp.Close()

	reader := bufio.NewReader(fp)
	data, _, oserr1 := reader.ReadLine()
	if oserr1 != nil {
		return uint32(0)
	}

	u, oserr2 := strconv.ParseUint(string(data), 10, 64)
	if oserr2 != nil {
		return uint32(0)
	}

	return uint32(u)
}

func (self *Mode) setCount(u uint32) *errors.Error {
	if u == 0 {
		os.Remove(self.FilePath)
		return nil
	}

	utils.MakeParentDir(self.FilePath, false)

	fp, oserr := os.OpenFile(self.FilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}
	defer fp.Close()

	fmt.Fprintf(fp, "%d", u)
	return nil
}

func (self *Mode) SetInteractMode() *errors.Error {
	n := self.getCount()
	return self.setCount(n + 1)
}

func (self *Mode) SetExpectMode() *errors.Error {
	n := self.getCount()
	if n > 0 {
		return self.setCount(n - 1)
	}
	return nil
}

func (self *Mode) IsInteractMode() bool {
	n := self.getCount()
	if n > 0 {
		return true
	} else {
		return false
	}
}

func (self *Mode) IsExpectMode() bool {
	n := self.getCount()
	if n <= 0 {
		return true
	} else {
		return false
	}
}

func (self *Mode) Dump() {
	fmt.Println("* Mode")
	fmt.Println("  FilePath:", self.FilePath)
}
