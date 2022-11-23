package record3

import (
	"discovery/config"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/proc"
	"discovery/utils"
	"os"
	"strings"
	"time"
)

func GetInputFilter() ([]FilterInterface, *errors.Error) {
	filter := []FilterInterface{}

	/* terminal input 옵션
	 * XXX: 순서 유지
	 */
	termcmd, err := NewTermCmd()
	if err != nil {
		return filter, err
	}
	filter = append(filter, termcmd)

	cmdcontrolBlock, err := NewCmdControl(constdef.CMD_CONTROL_MODE_BLOCK)
	if err != nil {
		return filter, err
	}
	filter = append(filter, cmdcontrolBlock)

	terminput, err := NewTermInput()
	if err != nil {
		return filter, err
	}
	filter = append(filter, terminput)

	/* send 기록
	 */
	recordInput, err := NewRecordInput()
	if err != nil {
		return filter, err
	}
	filter = append(filter, recordInput)

	cmdcontrolInteract, err := NewCmdControl(constdef.CMD_CONTROL_MODE_AUTO)
	if err != nil {
		return filter, err
	}
	filter = append(filter, cmdcontrolInteract)

	return filter, nil
}

func GetOutputFilter() ([]FilterInterface, *errors.Error) {
	filter := []FilterInterface{}

	/* output msg expect 기록
	 */
	recordOutput, err := NewRecordOutput()
	if err != nil {
		return filter, err
	}
	filter = append(filter, recordOutput)

	/* output msg 화면 출력
	 */
	termOutput, err := NewTermOutput()
	if err != nil {
		return filter, err
	}
	filter = append(filter, termOutput)

	return filter, nil
}

func findEnv(name string, cate []string) (*config.Env, *errors.Error) {
	record, err := NewRecord(name, cate)
	if err != nil {
		return nil, err
	}

	return record.FindEnv()
}

func connect(rid string, nodename string) (*proc.PtyProcess, *RecorderContext, *errors.Error) {
	if len(rid) == 0 || len(nodename) == 0 {
		return nil, nil, errors.New("Invalid arguments")
	}

	/* record 에서 env 찾아서 load함
	 */
	name, cate, err := utils.ParseRid(rid)
	if err != nil {
		return nil, nil, err
	}

	env, err := findEnv(name, cate)
	if err != nil {
		return nil, nil, err
	}

	connect, err := NewConnect2(nodename)
	if err != nil {
		return nil, nil, err
	}

	proc, promptstr, err := connect.Do2(env)
	if err != nil {
		return nil, nil, err
	}

	/* recorder context 생성
	 */
	context, err := NewRecorderContext(rid, nodename, connect.SessionName, env, proc)
	if err != nil {
		return nil, nil, err
	}
	context.LastPromptStr = promptstr

	/* 초기 record 기록
	 */
	comment, err := NewComment(fmt.Sprintf("* %s($<%s:ip>) 에 접속한다. 세션 ID: %s", nodename, nodename, connect.SessionName))
	if err != nil {
		return nil, nil, err
	}

	err = context.Logger.Write(comment.ToString(), 1)
	if err != nil {
		context.Close()
		return nil, nil, err
	}

	err = context.Logger.Write(connect.ToString(), 3)
	if err != nil {
		context.Close()
		return nil, nil, err
	}

	return proc, context, nil
}

func spawn(rid string, command string) (*proc.PtyProcess, *RecorderContext, *errors.Error) {
	if len(rid) == 0 || len(command) == 0 {
		return nil, nil, errors.New("Invalid arguments")
	}

	/* record 에서 env 찾아서 load함
	 */
	name, cate, err := utils.ParseRid(rid)
	if err != nil {
		return nil, nil, err
	}

	env, err := findEnv(name, cate)
	if err != nil {
		return nil, nil, err
	}

	/* spawn rcmd obj 생성
	 */
	spawn, err := NewSpawn2(command)
	if err != nil {
		return nil, nil, err
	}

	proc, err := spawn.Do2()
	if err != nil {
		return nil, nil, err
	}

	/* context 생성
	 */
	context, err := NewRecorderContext(rid, "", spawn.SessionName, env, proc)
	if err != nil {
		return nil, nil, err
	}

	comment, err := NewComment(fmt.Sprintf("* \"%s\" 를 실행했습니다. 세션 ID: %s", command, spawn.SessionName))
	if err != nil {
		return nil, nil, err
	}

	err = context.Logger.Write(comment.ToString(), 1)
	if err != nil {
		context.Close()
		return nil, nil, err
	}

	err = context.Logger.Write(spawn.ToString(), 3)
	if err != nil {
		context.Close()
		return nil, nil, err
	}

	return proc, context, nil
}

/* recording 시작
 */
func StartRecorder(arg *RecorderArg) *errors.Error {
	var proc *proc.PtyProcess = nil
	var context *RecorderContext = nil
	var err *errors.Error = nil

	switch strings.ToLower(arg.InitRcmd) {
	case "connect":
		proc, context, err = connect(arg.Rid, arg.InitRcmdArg)
		if err != nil {
			return err
		}
	case "spawn":
		proc, context, err = spawn(arg.Rid, arg.InitRcmdArg)
		if err != nil {
			return err
		}
	default:
		return errors.New("Invalid init cmd")
	}

	defer func() {
		proc.Stop()
		context.Close()
	}()

	inputFilter, err := GetInputFilter()
	if err != nil {
		return err
	}

	outputFilter, err := GetOutputFilter()
	if err != nil {
		return err
	}

	io, err := NewTermIO(proc, outputFilter, inputFilter)
	if err != nil {
		return err
	}

	/* main terminal Input/Output processing
	 */
	io.Start(context)

	return nil
}

/* rid에 해당하는 record 생성
 */
func CreateNewRecord(arg *RecorderArg) *errors.Error {
	name, cate, err := utils.ParseRid(arg.Rid)
	if err != nil {
		return err
	}

	recordPath, err := config.GetContentsRecordPath(name, cate)
	if err != nil {
		return err
	}

	/* record 파일 이미 존재하는지 check
	 */
	if _, goerr := os.Stat(recordPath); goerr == nil {
		return errors.New(recordPath + " record already exist")
	}

	/* record 파일 생성
	 * Environment 환경 설정 포함
	 */
	logger, err := NewRecorderLogger(cate, name)
	if err != nil {
		return err
	}
	defer logger.Close()

	// record head 생성
	t := time.Now()
	commentStr := fmt.Sprintf("Created by %s %s, %s", constdef.PRODUCT_NAME, constdef.PRODUCT_VERSION, t.Format(time.ANSIC))
	commentRcmd, err := NewComment(commentStr)
	if err != nil {
		return err
	}
	err = logger.Write(commentRcmd.ToString(), 1)
	if err != nil {
		return err
	}

	// record version 정보 기록
	versionRcmd, err := NewVersion2()
	if err != nil {
		return err
	}
	err = logger.Write(versionRcmd.ToString(), 2)
	if err != nil {
		return err
	}

	// environment 기록
	environmentRcmd, err := NewEnvironment2(arg.EnvId)
	if err != nil {
		return err
	}
	err = logger.Write(environmentRcmd.ToString(), 2)
	if err != nil {
		return err
	}

	if arg.CheckerFlag {
		/* checker 파일 생성
		 */
		checkerPath, err := config.GetContentsCheckerPath(name, cate)
		if err != nil {
			return err
		}

		checkerPath = checkerPath + ".sh"
		if _, goerr := os.Stat(checkerPath); goerr == nil {
			return nil
		}

		utils.MakeParentDir(checkerPath, false)
		fp, goerr := os.OpenFile(checkerPath, os.O_CREATE|os.O_WRONLY, 0755)
		if goerr != nil {
			e := errors.New(fmt.Sprintf("%s", goerr))
			fmt.Println("WARN: ", e.ToString(constdef.DEBUG), ", continue")
			return nil
		}
		defer fp.Close()

		fmt.Fprintf(fp, `#!/bin/bash

LOGDIR=$1
if test ! -e "$LOGDIR"; then
	echo "$LOGDIR doesn't exist, abort"
	exit 1
fi

##############################
# CHECK CODE BELOW
##############################


exit 0
`)
	}

	return nil
}
