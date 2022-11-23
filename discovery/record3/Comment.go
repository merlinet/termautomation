package record3

import (
	"discovery/errors"
	"discovery/fmt"
	"github.com/alecthomas/repr"
	"strings"
)

const CommentRcmdStr = ";"

type Comment struct {
	Name string `@COMMENT`
}

func NewComment(msg string) (*Comment, *errors.Error) {
	if len(msg) == 0 {
		return nil, errors.New("invalid msg string")
	}

	comment := Comment{
		Name: CommentRcmdStr + msg,
	}
	return &comment, nil
}

func (self *Comment) ToString() string {
	return fmt.Sprintf("%s", self.Name)
}

func (self *Comment) Prepare(context *ReplayerContext) *errors.Error {
	return nil
}

func (self *Comment) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("Invalid arguments").AddMsg(self.ToString())
	}

	comment := strings.TrimSpace(self.Name[1:])

	if len(comment) > 0 {
		commentType := comment[0]
		switch commentType {
		case '=', '-', '#', '%', '*', '_':
			comment2, err := context.ReplaceVariable(strings.TrimSpace(comment[1:]))
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}

			printflag, depthindent := context.GetResultOptions()

			/* check result 생성하고,
			 * step을 check result 생성 후 record result에 추가
			 */
			chkResult, err := NewCheckResult(string(commentType), comment2, printflag, depthindent)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
			step, err := NewStep(chkResult)
			if err != nil {
				return nil, err.AddMsg(self.ToString())
			}
			context.RecordResult.AddStep(step)
		}
	}

	return nil, nil
}

func (self *Comment) GetName() string {
	return self.Name
}

func (self *Comment) Dump() {
	repr.Println(self)
}
