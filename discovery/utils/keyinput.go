package utils

import (
	"bufio"
	"discovery/errors"
	"discovery/fmt"
	"github.com/creack/termios/raw"
	"os"
	"strings"
	"syscall"
	"unsafe"
)

func GetKeyInput(prompt string) (string, *errors.Error) {
	oldt, goerr := raw.TcGetAttr(uintptr(syscall.Stdin))
	if goerr != nil {
		return "", errors.New(fmt.Sprintf("%s", goerr))
	}

	// C code, newt.c_lflag &= ~(ECHO|ISIG);
	newt := *oldt
	newt.Lflag &^= (syscall.ISIG)
	goerr = raw.TcSetAttr(uintptr(syscall.Stdin), &newt)
	if goerr != nil {
		return "", errors.New(fmt.Sprintf("%s", goerr))
	}
	defer func() {
		raw.TcSetAttr(uintptr(syscall.Stdin), oldt)
	}()

	fmt.Printf("%s", prompt)

	reader := bufio.NewReader(os.Stdin)
	text, goerr := reader.ReadString('\n')
	if goerr != nil {
		return "", errors.New(fmt.Sprintf("%s", goerr))
	}

	return strings.TrimSpace(text), nil
}

const (
	KEY_ZERO      = 0x00
	KEY_CTRL_C    = 0x03
	KEY_TAB       = 0x09
	KEY_ENTER     = 0x0a
	KEY_CTRL_L    = 0x0c
	KEY_CTRL_U    = 0x15
	KEY_ESC       = 0x1b
	KEY_ESC_FUNC  = 0x5b
	KEY_ESC_FUNC2 = 0x33
	KEY_ESC_UP    = 0x41
	KEY_ESC_DOWN  = 0x42
	KEY_ESC_RIGHT = 0x43
	KEY_ESC_LEFT  = 0x44
	KEY_ESC_END   = 0x46
	KEY_ESC_HOME  = 0x48
	KEY_DEL       = 0x7e
	KEY_BACKSPACE = 0x7f
)

type WinSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

/* key 입력 history 처리
 */
type KeyInput struct {
	OldTermAttr *raw.Termios
	Reader      *bufio.Reader
	HistoryIdx  int
	History     [][]rune
	BufferIdx   int
	Buffer      []rune
	Prompt      string

	Winsize *WinSize // windows size
}

func NewKeyInput(prompt string) (*KeyInput, *errors.Error) {
	oldt, goerr := raw.TcGetAttr(uintptr(syscall.Stdin))
	if goerr != nil {
		return nil, errors.New(fmt.Sprintf("%s", goerr))
	}

	keyinput := KeyInput{
		OldTermAttr: oldt,
		Reader:      bufio.NewReader(os.Stdin),
		HistoryIdx:  0,
		History:     [][]rune{},
		BufferIdx:   0,
		Buffer:      []rune{},
		Prompt:      prompt,
	}

	return &keyinput, nil
}

func (self *KeyInput) SetInputFlag() *errors.Error {
	if self.OldTermAttr == nil {
		return errors.New("OldTermAttr is nil")
	}

	newt := *self.OldTermAttr
	newt.Lflag &^= (syscall.ECHO | syscall.ISIG | syscall.ICANON)
	goerr := raw.TcSetAttr(uintptr(syscall.Stdin), &newt)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}
	return nil
}

func (self *KeyInput) ReSetInputFlag() {
	if self.OldTermAttr != nil {
		raw.TcSetAttr(uintptr(syscall.Stdin), self.OldTermAttr)
	}
}

func (self *KeyInput) Close() {
	self.ReSetInputFlag()
}

func (self *KeyInput) GetWinSize() *errors.Error {
	ws := &WinSize{}
	retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return errors.New(fmt.Sprintf("retcode:", int(retCode), ", errno:", errno))
	}

	ws.Col -= 1
	self.Winsize = ws

	return nil
}

func (self *KeyInput) Input() (string, *errors.Error) {
	if self.Reader == nil {
		return "", errors.New("Reader is nil")
	}

	err := self.SetInputFlag()
	if err != nil {
		return "", err
	}
	defer self.ReSetInputFlag()

	fmt.Printf("%s", self.Prompt)

	for {
		err := self.GetWinSize()
		if err != nil {
			return "", err
		}

		b, _, goerr := self.Reader.ReadRune()
		if goerr != nil {
			return "", errors.New(fmt.Sprintf("%s", goerr))
		}

		//fmt.Printf("b --> %d 0x%02x, %c\n", b, b, b)

		switch b {
		case KEY_ZERO: // ignore
			continue

		case KEY_CTRL_C:
			self.Buffer = []rune{}
			return string(KEY_CTRL_C), nil

		case KEY_TAB: // ignore
			continue

		case KEY_ENTER:
			fmt.Println()
			msg := string(self.Buffer)
			if len(msg) > 0 {
				if len(self.History) == 0 || (len(self.History) > 0 && string(self.History[len(self.History)-1]) != msg) {
					self.History = append(self.History, self.Buffer)
				}
			}
			self.Reset()
			return msg, nil

		case KEY_CTRL_L:
			self.ResetLine()
			self.DrawLine()

		case KEY_CTRL_U:
			self.Reset()

		case KEY_ESC:
			c, _, goerr := self.Reader.ReadRune()
			if goerr != nil {
				return "", errors.New(fmt.Sprintf("%s", goerr))
			}

			//fmt.Printf("c --> %d 0x%02x, %c\n", c, c, c)

			if c == KEY_ESC_FUNC {
				d, _, goerr := self.Reader.ReadRune()
				if goerr != nil {
					return "", errors.New(fmt.Sprintf("%s", goerr))
				}

				//fmt.Printf("d --> %d 0x%02x, %c\n", d, d, d)

				switch d {
				case KEY_ESC_UP:
					self.UpHistory()
				case KEY_ESC_DOWN:
					self.DownHistory()
				case KEY_ESC_RIGHT:
					self.RightCursor()
				case KEY_ESC_LEFT:
					self.LeftCursor()
				case KEY_ESC_HOME:
					self.ResetLine()
					self.BufferIdx = 0
					self.DrawLine()
				case KEY_ESC_END:
					self.ResetLine()
					self.BufferIdx = len(self.Buffer)
					self.DrawLine()
				case KEY_ESC_FUNC2:
					e, _, goerr := self.Reader.ReadRune()
					if goerr != nil {
						return "", errors.New(fmt.Sprintf("%s", goerr))
					}

					//fmt.Printf("e --> %d 0x%02x, %c\n", e, e, e)

					switch e {
					case KEY_DEL:
						if len(self.Buffer) > 0 && self.BufferIdx >= 0 {
							if self.BufferIdx < len(self.Buffer) {
								self.Del()
							} else if self.BufferIdx == len(self.Buffer) {
								self.BackSpace()
							}
						}
					}
				}
			}

		case KEY_BACKSPACE:
			if len(self.Buffer) > 0 && self.BufferIdx > 0 {
				self.BackSpace()
			}

		default:
			if self.BufferIdx == len(self.Buffer) {
				self.Append(b)
			} else {
				self.Insert(b)
			}
		}
	}
}

func (self *KeyInput) UpHistory() {
	if len(self.History) == 0 {
		return
	}

	if self.HistoryIdx > 0 {
		self.HistoryIdx--

		self.ResetLine()
		self.Buffer = []rune(self.History[self.HistoryIdx])
		self.BufferIdx = len(self.Buffer)
		self.DrawLine()
	}
}

func (self *KeyInput) DownHistory() {
	if len(self.History) == 0 {
		return
	}

	if self.HistoryIdx < len(self.History)-1 {
		self.HistoryIdx++

		self.ResetLine()
		self.Buffer = []rune(self.History[self.HistoryIdx])
		self.BufferIdx = len(self.Buffer)
		self.DrawLine()
	}
}

func (self *KeyInput) LeftCursor() {
	if len(self.Buffer) == 0 {
		return
	}

	self.ResetLine()
	if self.BufferIdx > 0 {
		self.BufferIdx--
	}
	self.DrawLine()
}

func (self *KeyInput) RightCursor() {
	if len(self.Buffer) == 0 {
		return
	}

	self.ResetLine()
	if self.BufferIdx < len(self.Buffer) {
		self.BufferIdx++
	}
	self.DrawLine()
}

/* windows col size를 계산해서 화면서 출력할 buffer chunk 반환
 */
func (self *KeyInput) CalculateColBuffer() ([]rune, int) {
	bufferIndexSize := int(len(self.Buffer) / int(self.Winsize.Col))
	bulkIdx := int(self.BufferIdx / int(self.Winsize.Col))
	idx := int(self.BufferIdx % int(self.Winsize.Col))

	if bulkIdx > bufferIndexSize {
		return []rune{}, -1
	}

	start := bulkIdx * int(self.Winsize.Col)
	end := (bulkIdx + 1) * int(self.Winsize.Col)
	if end > len(self.Buffer) {
		end = len(self.Buffer)
	}
	buf := self.Buffer[start:end]

	return buf, idx
}

func (self *KeyInput) ResetLine() {
	if len(self.Buffer) == 0 {
		return
	}

	buf, idx := self.CalculateColBuffer()
	if idx < 0 {
		return
	}

	fmt.Printf("%s", string(buf[idx:]))
	Backspace(buf)
}

func (self *KeyInput) DrawLine() {
	if len(self.Buffer) == 0 {
		return
	}

	buf, idx := self.CalculateColBuffer()
	if idx < 0 {
		return
	}

	fmt.Printf("%s", string(buf))
	Backmove(buf)
	fmt.Printf("%s", string(buf[:idx]))
}

func (self *KeyInput) Append(b rune) {
	self.ResetLine()
	self.Buffer = append(self.Buffer, b)
	self.BufferIdx = len(self.Buffer)
	self.DrawLine()
	self.HistoryIdx = len(self.History)
}

func (self *KeyInput) Insert(b rune) {
	if len(self.Buffer) == 0 || self.BufferIdx < 0 || self.BufferIdx > len(self.Buffer) {
		return
	}

	self.ResetLine()
	buffer := []rune{}
	buffer = append(buffer, self.Buffer[:self.BufferIdx]...)
	buffer = append(buffer, b)
	buffer = append(buffer, self.Buffer[self.BufferIdx:]...)
	self.Buffer = buffer
	self.BufferIdx++
	self.DrawLine()
	self.HistoryIdx = len(self.History)
}

func (self *KeyInput) Del() {
	if len(self.Buffer) == 0 || self.BufferIdx < 0 || self.BufferIdx > len(self.Buffer) {
		return
	}

	self.ResetLine()
	buffer := []rune{}
	buffer = append(buffer, self.Buffer[:self.BufferIdx]...)
	buffer = append(buffer, self.Buffer[self.BufferIdx+1:]...)
	self.Buffer = buffer
	self.DrawLine()
	self.HistoryIdx = len(self.History)
}

func (self *KeyInput) BackSpace() {
	if len(self.Buffer) == 0 || self.BufferIdx <= 0 || self.BufferIdx > len(self.Buffer) {
		return
	}

	self.ResetLine()
	self.BufferIdx--
	buffer := []rune{}
	buffer = append(buffer, self.Buffer[:self.BufferIdx]...)
	buffer = append(buffer, self.Buffer[self.BufferIdx+1:]...)
	self.Buffer = buffer
	self.DrawLine()
	self.HistoryIdx = len(self.History)
}

func (self *KeyInput) Reset() {
	self.ResetLine()
	self.Buffer = []rune{}
	self.BufferIdx = 0
	self.DrawLine()
	self.HistoryIdx = len(self.History)
}

/* backspace 처리
 */
func Backspace(buffer []rune) {
	for i := len(buffer) - 1; i >= 0; i-- {
		size := len([]byte(string(buffer[i])))
		if size == 1 {
			fmt.Printf("\b \b")
		} else {
			fmt.Printf("\b\b \b")
		}
	}
}

/* back move 처리
 */
func Backmove(buffer []rune) {
	for i := len(buffer) - 1; i >= 0; i-- {
		size := len([]byte(string(buffer[i])))
		if size == 1 {
			fmt.Printf("\b")
		} else {
			fmt.Printf("\b\b")
		}
	}
}
