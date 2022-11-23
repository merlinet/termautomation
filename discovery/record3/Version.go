package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
)

const VersionRcmdStr = "version"

type Version struct {
	Name    string  `@"version"`
	Version float64 `@NUMBER`
}

func NewVersion(text string) (*Version, *errors.Error) {
	target, err := NewStruct(text, &Version{})
	if err != nil {
		return nil, err
	}
	return target.(*Version), nil
}

func NewVersion2() (*Version, *errors.Error) {
	version := Version{
		Name:    VersionRcmdStr,
		Version: constdef.RECORD_VERSION,
	}

	return &version, nil
}

func (self *Version) ToString() string {
	return fmt.Sprintf("%s %.0f", self.Name, self.Version)
}

func (self *Version) Prepare(context *ReplayerContext) *errors.Error {
	if self.Version != constdef.RECORD_VERSION {
		return errors.New(fmt.Sprintf("This only support Record Version %.0f", constdef.RECORD_VERSION)).AddMsg(self.ToString())
	}

	return nil
}

func (self *Version) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("invalid arguments").AddMsg(self.ToString())
	}

	context.RecordVersion = uint32(self.Version)
	return nil, nil
}

func (self *Version) GetName() string {
	return self.Name
}

func (self *Version) Dump() {
	repr.Println(self)
}
