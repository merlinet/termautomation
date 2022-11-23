package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const SpawnRcmdStr = "spawn"

type Spawn struct {
	Name        string `@"spawn"`
	Command     string `@STRING`
	SessionName string `@IDENT`
}

func NewSpawn(text string) (*Spawn, *errors.Error) {
	target, err := NewStruct(text, &Spawn{})
	if err != nil {
		return nil, err
	}
	return target.(*Spawn), nil
}

func NewSpawn2(command string) (*Spawn, *errors.Error) {
	if len(command) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	spawn := Spawn{
		Name:        SpawnRcmdStr,
		Command:     utils.Quote(command),
		SessionName: config.GetSessionId(""),
	}
	return &spawn, nil
}

func (self *Spawn) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.Command, self.SessionName)
}

func (self *Spawn) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Spawn) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	_, ok := context.SessionMap[self.SessionName]
	if ok {
		return nil, errors.New(fmt.Sprintf("%s session already made", self.SessionName)).AddMsg(self.ToString())
	}

	command, err := context.ReplaceVariable(utils.Unquote(self.Command))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	proc, err := doSpawn(command)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	sessionnode, err := NewSessionNode(context.LogDir, self.SessionName, proc, nil)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	context.SessionMap[self.SessionName] = sessionnode

	return nil, nil
}

func (self *Spawn) Do2() (*proc.PtyProcess, *errors.Error) {
	proc, err := doSpawn(utils.Unquote(self.Command))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}
	return proc, nil
}

func doSpawn(command string) (*proc.PtyProcess, *errors.Error) {
	if len(command) == 0 {
		return nil, errors.New("Invalid command string")
	}

	proc, err := proc.NewPtyProcess(command, constdef.DEFAULT_CHARACTER_SET, constdef.DEFAULT_EOL)
	if err != nil {
		return nil, err
	}

	err = proc.Start()
	if err != nil {
		return nil, err
	}

	return proc, nil
}

func (self *Spawn) GetName() string {
	return self.Name
}

func (self *Spawn) Dump() {
	repr.Println(self)
}
