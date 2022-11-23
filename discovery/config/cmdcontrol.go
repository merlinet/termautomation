package config

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"github.com/go-ini/ini"
	"regexp"
)

type CCElement struct {
	Cmd string
	Re  *regexp.Regexp
}

func NewCCElement(cmd string) (*CCElement, *errors.Error) {
	if len(cmd) == 0 {
		return nil, errors.New("invalid cmd")
	}

	re, oserr := regexp.Compile(cmd)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	ccelement := CCElement{
		Cmd: cmd,
		Re:  re,
	}

	return &ccelement, nil
}

func (self *CCElement) Match(cmd string) bool {
	return self.Re.MatchString(cmd)
}

func (self *CCElement) Dump() {
	fmt.Printf("    Cmd:%s, ", self.Cmd)
	fmt.Println("    Re:", self.Re)
}

type CmdControl struct {
	CmdControlConfPath string
	BlockList          []*CCElement
	InteractList       []*CCElement
	IgnoreStrList      []*CCElement
}

func NewCmdControl() (*CmdControl, *errors.Error) {
	path, err := GetDiscoveryEtcCmdControlConfPath()
	if err != nil {
		return nil, err
	}

	cmdControl := CmdControl{
		CmdControlConfPath: path,
		BlockList:          make([]*CCElement, 0),
		InteractList:       make([]*CCElement, 0),
		IgnoreStrList:      make([]*CCElement, 0),
	}

	err = cmdControl.load()
	if err != nil {
		return nil, err
	}

	return &cmdControl, nil
}

func (self *CmdControl) load() *errors.Error {
	conf, goerr := ini.LoadSources(ini.LoadOptions{AllowBooleanKeys: true, IgnoreInlineComment: true}, self.CmdControlConfPath)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	blocklist := conf.Section("BLOCK").KeyStrings()
	addList(blocklist, &self.BlockList)

	interactlist := conf.Section("INTERACT").KeyStrings()
	addList(interactlist, &self.InteractList)

	ignorerecordlist := conf.Section("IGNORE").KeyStrings()
	addList(ignorerecordlist, &self.IgnoreStrList)

	return nil
}

func addList(cmdlist []string, list *[]*CCElement) {
	for _, cmd := range cmdlist {
		cce, err := NewCCElement(cmd)
		if err != nil {
			fmt.Println("WARN:", err.ToString(constdef.DEBUG), ", continue")
			continue
		}
		*list = append(*list, cce)
	}
}

func matchCmd(cmd string, list []*CCElement) bool {
	for _, cce := range list {
		if cce.Match(cmd) {
			return true
		}
	}

	return false
}

func (self *CmdControl) IsBlockCmd(cmd string) bool {
	return matchCmd(cmd, self.BlockList)
}

func (self *CmdControl) IsInteractCmd(cmd string) bool {
	return matchCmd(cmd, self.InteractList)
}

func (self *CmdControl) IsIgnoreStr(cmd string) bool {
	return matchCmd(cmd, self.IgnoreStrList)
}

func (self *CmdControl) Dump() {
	fmt.Println("* CmdControl")
	fmt.Println(" CmdControlConfPath:", self.CmdControlConfPath)
	fmt.Println(" - BlockList:")
	dumpList(self.BlockList)
	fmt.Println(" - InteractList:")
	dumpList(self.InteractList)
	fmt.Println(" - IgnoreStrList:")
	dumpList(self.IgnoreStrList)
}

func dumpList(list []*CCElement) {
	for _, cce := range list {
		cce.Dump()
	}
}
