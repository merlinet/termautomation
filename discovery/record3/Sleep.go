package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
	"time"
)

const SleepRcmdStr = "sleep"

type Sleep struct {
	Name             string  `@"sleep"`
	SleepMilliSecond float64 `@NUMBER`
}

func NewSleep(text string) (*Sleep, *errors.Error) {
	target, err := NewStruct(text, &Sleep{})
	if err != nil {
		return nil, err
	}
	return target.(*Sleep), nil
}

func (self *Sleep) ToString() string {
	return fmt.Sprintf("%s %d", self.Name, int(self.SleepMilliSecond))
}

func (self *Sleep) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Sleep) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	exitChan := make(chan bool)

	defer func() {
		close(exitChan)
		if context.OutputPrintFlag {
			fmt.Println("")
		}
	}()

	go func() {
		if context.OutputPrintFlag {
			fmt.Printf("\n>>> Sleep %.1f Millisecond ", self.SleepMilliSecond)
		}
		for {
			select {
			case <-exitChan:
				return
			default:
				if context.OutputPrintFlag {
					fmt.Printf(".")
				}
				time.Sleep(time.Second * 1)
			}
		}
	}()

	time.Sleep(time.Millisecond * time.Duration(self.SleepMilliSecond))

	return nil, nil
}

func (self *Sleep) GetName() string {
	return self.Name
}

func (self *Sleep) Dump() {
	repr.Println(self)
}
