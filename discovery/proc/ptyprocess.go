package proc

import (
	"bufio"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"github.com/creack/pty"
	"github.com/lunixbochs/vtclean"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

/*
#include <sys/ioctl.h>
*/
import "C"

const (
	LINE_TYPE_OUTPUT_LINE = 1 + iota
	LINE_TYPE_PROMPT
	LINE_TYPE_SSHAUTH
	LINE_TYPE_MORE

	MATCH_TYPE_RE
	MATCH_TYPE_STR_CONTAIN
	MATCH_TYPE_STR_EXACT
)

type LineMatch struct {
	LineType  uint8
	MatchType uint8
	Re        *regexp.Regexp // MATCH_TYPE_RE
	Str       string         // MATCH_TYPE_STR_COTAIN, MATCH_TYPE_STR_EXACT
}

type PtyProcess struct {
	CommandStr    string
	ExecCmd       *exec.Cmd
	Fp            *os.File
	Reader        *bufio.Reader
	OutputChannel chan string
	CharacterSet  string
	Eol           string
}

func NewPtyProcess(command string, characterSet string, eol string) (*PtyProcess, *errors.Error) {
	varArgs := strings.Split(command, " ")
	if len(varArgs) <= 0 {
		return nil, errors.New("Invalid command")
	}

	charSet := constdef.DEFAULT_CHARACTER_SET
	switch strings.ToLower(strings.TrimSpace(characterSet)) {
	case "euckr":
		charSet = "euckr"
	case constdef.DEFAULT_CHARACTER_SET:
	case "":
	default:
		return nil, errors.New("Invalid characterSet. euckr, utf8 can available")
	}

	switch eol {
	case constdef.EOL_CR:
		eol = "\r"
	case constdef.EOL_LF:
		eol = "\n"
	case constdef.EOL_CRLF:
		eol = "\r\n"
	default:
		return nil, errors.New("invalid eol argument")
	}

	ptyprocess := PtyProcess{
		CommandStr:    command,
		ExecCmd:       exec.Command(varArgs[0], varArgs[1:]...),
		Fp:            nil,
		OutputChannel: make(chan string),
		CharacterSet:  charSet,
		Eol:           eol,
	}

	return &ptyprocess, nil
}

func (self *PtyProcess) Start() *errors.Error {
	wz, goerr := pty.GetsizeFull(os.Stdout)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	fp, goerr := pty.StartWithSize(self.ExecCmd, wz)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	self.Fp = fp
	self.Reader = bufio.NewReader(fp)

	go func() {
		defer close(self.OutputChannel)

		for {
			line, _, err := self.Reader.ReadLine()
			if err != nil {
				return
			}
			self.OutputChannel <- string(line)
		}
	}()

	return nil
}

func (self *PtyProcess) Read(matchtable []*LineMatch, timeout time.Duration) (string, uint8, *errors.Error) {
	timeoutCount := timeout

	for {
		select {
		case line, ok := <-self.OutputChannel:
			if !ok {
				return "", 0, errors.New("OutputChannel has closed.")
			}
			return vtclean.Clean(line, false), LINE_TYPE_OUTPUT_LINE, nil
		case <-time.After(time.Millisecond * time.Duration(constdef.DEFAULT_EXPECT_TIMEOUT_STEP)):
			unread := self.Reader.Buffered()
			peekBuf, err := self.Reader.Peek(unread)
			if err != nil {
				return "", 0, errors.New(fmt.Sprintf("%s", err))
			}
			line := vtclean.Clean(string(peekBuf), false)

			for _, matchtable := range matchtable {
				switch matchtable.MatchType {
				case MATCH_TYPE_RE:
					if matchtable.Re == nil {
						return "", 0, errors.New("regexp is nil")
					}
					if matchtable.Re.Match([]byte(line)) {
						return line, matchtable.LineType, nil
					}
				case MATCH_TYPE_STR_EXACT:
					if line == matchtable.Str {
						return line, matchtable.LineType, nil
					}
				case MATCH_TYPE_STR_CONTAIN:
					if strings.Contains(line, matchtable.Str) {
						return line, matchtable.LineType, nil
					}
				default:
					return "", 0, errors.New("invalid prompt match type")
				}
			}

			if timeout > 0 {
				timeoutCount -= time.Duration(constdef.DEFAULT_EXPECT_TIMEOUT_STEP)
				if timeoutCount <= 0 {
					return "", 0, errors.New("timeout")
				}
			}
		}
	}
}

func (self *PtyProcess) Write(rawMsg string) *errors.Error {
	msg := rawMsg
	switch self.CharacterSet {
	case "euckr":
		got, _, oserr := transform.String(korean.EUCKR.NewEncoder(), msg)
		if oserr == nil {
			msg = got
		} else {
			fmt.Println("WARN:", errors.New(fmt.Sprintf("%s", oserr)).ToString(constdef.DEBUG))
		}
	case constdef.DEFAULT_CHARACTER_SET:
	default:
	}

	_, goerr := self.Fp.Write([]byte(msg))
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	return nil
}

func (self *PtyProcess) GetReceivedBufferSize() (int, *errors.Error) {
	if self.Fp == nil {
		return -1, errors.New("Invalid fp")
	}

	var f int
	FIONREAD := int(C.FIONREAD)
	_, _, oserrno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(self.Fp.Fd()), uintptr(FIONREAD), uintptr(unsafe.Pointer(&f)))
	if oserrno != 0 {
		return -1, errors.New(fmt.Sprintf("errno %d", oserrno))
	}

	return f, nil
}

func (self *PtyProcess) Stop() {
	self.Fp.Close()
	self.ExecCmd.Process.Kill()
	self.ExecCmd.Wait()
}
