package record3

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"os"
	"time"
)

type TermIO struct {
	ExitFlag    bool
	ExitChannel chan bool

	Proc *proc.PtyProcess

	InputChannel  chan string
	OutputChannel chan string

	OutputFilter []FilterInterface
	InputFilter  []FilterInterface

	KeyInputHandler *utils.KeyInput
}

func NewTermIO(proc *proc.PtyProcess, outputFilter, inputFilter []FilterInterface) (*TermIO, *errors.Error) {
	if proc == nil {
		return nil, errors.New("Invalid arguments")
	}

	key, err := utils.NewKeyInput("")
	if err != nil {
		return nil, err
	}

	termio := TermIO{
		ExitFlag:    false,
		ExitChannel: make(chan bool),

		Proc: proc,

		InputChannel:  make(chan string),
		OutputChannel: proc.OutputChannel,

		OutputFilter: outputFilter,
		InputFilter:  inputFilter,

		KeyInputHandler: key,
	}

	return &termio, nil
}

func (self *TermIO) InputMsg(msg string) {
	self.InputChannel <- msg
}

func (self *TermIO) processKeyInput() {
	defer func() {
		self.ExitChannel <- true
		self.ExitFlag = true
	}()

	for self.ExitFlag != true {
		msg, err := self.KeyInputHandler.Input()
		if err != nil {
			fmt.Println("ERR:", err.ToString(constdef.DEBUG))
			return
		}
		self.InputMsg(msg)
	}
}

func (self *TermIO) processIO(context *RecorderContext) {
	defer func() {
		self.ExitFlag = true
	}()

	if context == nil {
		return
	}

	for self.ExitFlag != true {
		select {
		case msg, ok := <-self.InputChannel:
			if !ok {
				return
			}

			/* input msg 처리
			 */
			for _, filter := range self.InputFilter {
				continueFlag := filter.Do(context, msg, constdef.IO_SELECTER_INPUT)
				if continueFlag != true {
					break
				}
			}

		case msg, ok := <-self.OutputChannel:
			if !ok {
				/* 종료전 expect 모드 설정
				 */
				if context.Mode == constdef.MODE_INTERACT ||
					context.ModeChange == constdef.MODE_INTERACT {

					err := context.GlobalMode.SetExpectMode()
					if err != nil {
						fmt.Println(err.ToString(constdef.DEBUG))
					}
				}

				fmt.Println("ERR: proc output channel has closed.")
				self.KeyInputHandler.Close()
				os.Exit(0)
				return
			}

			/* Terminal output msg 처리
			 */
			for _, filter := range self.OutputFilter {
				continueFlag := filter.Do(context, msg, constdef.IO_SELECTER_OUTPUT)
				if continueFlag != true {
					break
				}
			}

		case <-time.After(time.Millisecond * constdef.OUTPUT_TIMEOUT_MILLISECOND):
			/* timeout 발생시 처리
			 */
			for _, filter := range self.OutputFilter {
				continueFlag := filter.Do(context, "", constdef.IO_SELECTER_TIMEOUT)
				if continueFlag != true {
					break
				}
			}

		case <-self.ExitChannel:
			return
		}
	}
}

func (self *TermIO) Start(context *RecorderContext) {
	go self.processKeyInput()

	self.processIO(context)
}
