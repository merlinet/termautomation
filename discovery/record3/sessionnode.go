package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
)

type SessionNode struct {
	SessionName  string
	Proc         *proc.PtyProcess
	Node         *config.Node // spawn인 경우 nil
	LogFlag      bool
	LogPrefix    string
	LogPath      string
	Logger       *ReplayerLogger
	LogSendCount uint /* 로그 prefix 처리에 사용 */
}

func NewSessionNode(logdir, sessionname string, procPtr *proc.PtyProcess, node *config.Node) (*SessionNode, *errors.Error) {
	if len(logdir) == 0 || len(sessionname) == 0 || procPtr == nil {
		return nil, errors.New("invalid arguments")
	}

	logpath := fmt.Sprintf("%s/%s.log", logdir, sessionname)
	logger, err := NewReplayerLogger(logpath)
	if err != nil {
		return nil, err
	}

	sessionnode := SessionNode{
		SessionName:  sessionname,
		Proc:         procPtr,
		Node:         node,
		LogFlag:      false,
		LogPath:      logpath,
		Logger:       logger,
		LogSendCount: 0,
	}

	return &sessionnode, nil
}

func (self *SessionNode) Close() {
	if self.Proc != nil {
		self.Proc.Stop()
	}

	if self.Logger != nil {
		self.Logger.Close()
	}
}
