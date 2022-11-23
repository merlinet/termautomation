/* Breaking Point restful api client 처리
 */
package record3

import (
	"discovery/errors"
	"discovery/utils"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	MODEL_TYPE_NORMAL = iota
	MODEL_TYPE_RFC2544

	RESULT_PASSED
	RESULT_FAILED
	RESULT_CANCELED
	RESULT_COMPLETED
	RESULT_INCOMPLETE
	RESULT_ERROR
	RESULT_UNKNOWN
)

type BPClient struct {
	RestClient

	Username string
	Password string

	Slot     int
	Portlist []int

	RunId map[string]string // run test 하면 append됨, test run 정보 확인시 갱신됨

	/* 시그널 제어
	 */
	EndingSignal chan os.Signal
	Done         chan bool
}

func NewBPClient(protocol, host string, port int, username, password string, slot int,
	portlist []int, apiPath string) (*BPClient, *errors.Error) {

	if len(host) == 0 || len(username) == 0 || len(password) == 0 ||
		len(portlist) == 0 || len(apiPath) == 0 {

		return nil, errors.New("invalid argument")
	}

	restclient, err := NewRestClient(protocol, host, port, apiPath)
	if err != nil {
		return nil, err
	}

	bpclient := BPClient{
		RestClient: *restclient,

		Username: username,
		Password: password,

		Slot:     slot,
		Portlist: portlist,

		RunId: make(map[string]string),
	}

	return &bpclient, nil
}

/* bp login 처리
 */
func (self *BPClient) Login() *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"username": self.Username,
		"password": self.Password,
	}

	success, _, bodyBytes, err := self.RestClient.Request("login", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	/* test 실행중 발생하는 interrupt 처리
	 * INT, TERM, QUIT 등 이 발생하면 stop하고 port unresove 처리함
	 */
	self.EndingSignal = make(chan os.Signal, 1)
	self.Done = make(chan bool, 1)

	/* signal 처리
	 */
	signal.Notify(self.EndingSignal, syscall.SIGINT, syscall.SIGTERM,
		syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGQUIT)
	go func() {
		select {
		case sig := <-self.EndingSignal:
			fmt.Println("-> signal:", sig)
			self.Close()
		case <-self.Done:
		}
	}()

	return nil
}

/* bp port reserve 처리
 */
func (self *BPClient) Reserveports() *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"slot":     self.Slot,
		"portList": self.Portlist,
	}

	success, _, bodyBytes, err := self.RestClient.Request("reserveports", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* bp port unreserve 처리
 */
func (self *BPClient) Unreserveports() *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"slot":     self.Slot,
		"portList": self.Portlist,
	}

	success, _, bodyBytes, err := self.RestClient.Request("unreserveports", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* bp test 실행
 */
func (self *BPClient) Runtest(modelname string, modeltype int) *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	if len(modelname) == 0 {
		return errors.New("invalid arguments")
	}

	/* 중복 시행 검사
	 */
	if _, ok := self.RunId[modelname]; ok {
		return errors.New(fmt.Sprintf("'%s' already ran", modelname))
	}

	/* bp test 모델 선택
	 * normal
	 * rfc2544
	 */
	switch modeltype {
	case MODEL_TYPE_NORMAL:
		err := self.SetNormalTest(modelname)
		if err != nil {
			return err
		}
	case MODEL_TYPE_RFC2544:
		err := self.SetRFC(modelname)
		if err != nil {
			return err
		}
	default:
		return errors.New("invalid model type")
	}

	reqData := map[Void]Void{
		"modelname": modelname,
	}

	success, _, bodyBytes, err := self.RestClient.Request("runtest", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	/* response body에서 testid 값 축출
	 */
	value, err := GetBodyValue(bodyBytes, "testid")
	if err != nil {
		return err
	}

	testid := fmt.Sprintf("%v", value)
	if f, ok := value.(float64); ok {
		testid = strconv.FormatFloat(f, 'f', -1, 64)
	}

	self.RunId[modelname] = testid
	return nil
}

func (self *BPClient) RuntestWait(modelname string, componentName string,
	modeltype int, waitInterval int, context *ReplayerContext) (int, *errors.Error) {

	if context == nil {
		return RESULT_UNKNOWN, errors.New("invalid arguments")
	}

	err := self.Runtest(modelname, modeltype)
	if err != nil {
		return RESULT_UNKNOWN, err
	}

	for {
		progress, err := self.GetRTS(modelname)
		if err != nil {
			return RESULT_UNKNOWN, err
		}

		/* step 생성
		 */
		printflag, depthindent := context.GetResultOptions()

		stepMsg := ""
		if len(componentName) > 0 {
			stepMsg = fmt.Sprintf(" -> %s(%s), Progress %.02f%%", modelname, componentName, progress)
		} else {
			stepMsg = fmt.Sprintf(" -> %s, Progress %.02f%%", modelname, progress)
		}

		chkres, err := NewCheckResult("*", stepMsg, printflag, depthindent)
		if err != nil {
			return RESULT_UNKNOWN, err
		}

		step, err := NewStep(chkres)
		if err != nil {
			return RESULT_UNKNOWN, err
		}
		context.RecordResult.AddStep(step)

		/* 진행율 검사
		 */
		if progress >= 100 {
			break
		}

		if waitInterval > 10 {
			time.Sleep(time.Second * time.Duration(waitInterval))
		} else {
			time.Sleep(time.Second * 10)
		}
	}

	/* result 가 passed, fail 나올때까지 기다림
	 */
	for {
		result, err := self.GetTestResult(modelname)
		if err != nil {
			return RESULT_UNKNOWN, err
		}

		switch result {
		case RESULT_PASSED, RESULT_FAILED, RESULT_CANCELED, RESULT_COMPLETED, RESULT_ERROR:
			return result, nil
		case RESULT_INCOMPLETE:
			time.Sleep(time.Second * 1)
		default:
			return RESULT_UNKNOWN, nil
		}
	}
}

func (self *BPClient) RuntestExportResult(modelname string, modeltype int, individualComponent bool,
	exportResult bool, context *ReplayerContext) ([]string, *errors.Error) {

	resOut := []string{}

	if context == nil {
		return resOut, errors.New("invalid arguments")
	}

	/* MODEL_TYPE_RFC2544 이거나
	 * MODEL_TYPE_NORMAL && individualComponent false
	 */
	if modeltype == MODEL_TYPE_RFC2544 ||
		(modeltype == MODEL_TYPE_NORMAL && individualComponent == false) {

		result, err := self.RuntestWait(modelname, "", modeltype, 10, context)
		if err != nil {
			return resOut, err
		}

		switch result {
		case RESULT_PASSED:
			resOut = append(resOut, modelname+": passed")
		case RESULT_FAILED:
			resOut = append(resOut, modelname+": failed")
		case RESULT_CANCELED:
			resOut = append(resOut, modelname+": canceled")
		case RESULT_COMPLETED:
			resOut = append(resOut, modelname+": completed")
		case RESULT_INCOMPLETE:
			resOut = append(resOut, modelname+": incomplete")
		case RESULT_ERROR:
			resOut = append(resOut, modelname+": error")
		case RESULT_UNKNOWN:
			resOut = append(resOut, modelname+": unknown")
		default:
			resOut = append(resOut, modelname+": unknown default")
		}

		if exportResult {
			downloadPath := fmt.Sprintf("%s/%s.pdf", context.LogDir, modelname)
			err = self.ExportTestResult(modelname, downloadPath)
			if err != nil {
				return resOut, err
			}
		}

		err = self.Stoptest([]string{modelname})
		if err != nil {
			return resOut, err
		}

		return resOut, nil
	}

	/* MODEL_TYPE_NORMAL && individualComponent true
	 */
	err := self.SetNormalTest(modelname)
	if err != nil {
		return resOut, err
	}

	componentNames, err := self.GetComponentNames(modelname)
	if err != nil {
		return resOut, err
	}

	for id, name := range componentNames {
		/* 진행 component 외 다른 component는 disable
		 */
		for id2, _ := range componentNames {
			if id2 == id {
				continue
			}
			err := self.ModifyNormalTest(id2, "active", false)
			if err != nil {
				return resOut, err
			}
		}

		err := self.ModifyNormalTest(id, "active", true)
		if err != nil {
			return resOut, err
		}

		err = self.SaveNormalTest(modelname)
		if err != nil {
			return resOut, err
		}

		/* modelname, componentid step 생성
		 */
		printflag, depthindent := context.GetResultOptions()
		stepMsg := fmt.Sprintf(" %s, %s is enabled", modelname, name)
		chkres, err := NewCheckResult("*", stepMsg, printflag, depthindent)
		if err != nil {
			return resOut, err
		}

		step, err := NewStep(chkres)
		if err != nil {
			return resOut, err
		}
		context.RecordResult.AddStep(step)

		/* model 실행
		 */
		result, err := self.RuntestWait(modelname, name, modeltype, 10, context)
		if err != nil {
			return resOut, err
		}

		switch result {
		case RESULT_PASSED:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": passed")
		case RESULT_FAILED:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": failed")
		case RESULT_CANCELED:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": canceled")
		case RESULT_COMPLETED:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": completed")
		case RESULT_INCOMPLETE:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": incomplete")
		case RESULT_ERROR:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": error")
		case RESULT_UNKNOWN:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": unknown")
		default:
			resOut = append(resOut, fmt.Sprintf("%s(%s)", modelname, name)+": unknown default")
		}

		if exportResult {
			downloadPath := fmt.Sprintf("%s/%s-%s.pdf", context.LogDir, modelname, name)
			err = self.ExportTestResult(modelname, downloadPath)
			if err != nil {
				return resOut, err
			}
		}

		err = self.Stoptest([]string{modelname})
		if err != nil {
			return resOut, err
		}
	}

	return resOut, nil
}

/* 실행 취소
 */
func (self *BPClient) Stoptest(modelList []string) *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	stopList := make(map[string]string)
	if len(modelList) > 0 {
		for _, name := range modelList {
			if runid, ok := self.RunId[name]; ok {
				stopList[name] = runid
			}
		}
	} else {
		stopList = self.RunId
	}

	for modelname, runid := range stopList {
		reqData := map[Void]Void{
			"testid": runid,
		}

		success, _, bodyBytes, err := self.RestClient.Request("stoptest", reqData)
		if err != nil {
			return err
		}

		if !success {
			errmsg, err := GetBodyError(bodyBytes)
			if err != nil {
				return err
			}
			return errors.New(errmsg)
		}

		/* result 가 canceled 나올때까지 대기
		 */
	out:
		for {
			result, err := self.GetTestResult(modelname)
			if err != nil {
				return err
			}

			switch result {
			case RESULT_PASSED, RESULT_FAILED, RESULT_CANCELED, RESULT_COMPLETED, RESULT_ERROR:
				break out
			case RESULT_INCOMPLETE:
				time.Sleep(time.Second * 1)
			default:
				return errors.New("invalid test result value")
			}
		}

		delete(self.RunId, modelname)
	}

	return nil
}

/* 테스트 실행 상태 확인
 */
func (self *BPClient) GetTestResult(modelname string) (int, *errors.Error) {
	if self.RestClient.Rest == nil {
		return -1, errors.New("bp client doesn't initialize")
	}

	runid, ok := self.RunId[modelname]
	if !ok {
		return -1, errors.New(fmt.Sprintf("'%s' is not running modlename", modelname))
	}

	reqData := map[Void]Void{
		"runid": runid,
	}

	success, _, bodyBytes, err := self.RestClient.Request("gettestresult", reqData)
	if err != nil {
		return -1, err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return -1, err
		}
		return -1, errors.New(errmsg)
	}

	result, err := GetBodyResult(bodyBytes)
	if err != nil {
		return -1, err
	}

	return result, nil
}

/* 테스트 progress 퍼센트 얻음
 */
func (self *BPClient) GetRTS(modelname string) (float64, *errors.Error) {
	if self.RestClient.Rest == nil {
		return -1, errors.New("bp client doesn't initialize")
	}

	if len(modelname) == 0 {
		return -1, errors.New("invalid argument")
	}

	runid, ok := self.RunId[modelname]
	if !ok {
		return -1, errors.New(fmt.Sprintf("'%s' is not running modlename", modelname))
	}

	reqData := map[Void]Void{
		"runid": runid,
	}

	success, _, bodyBytes, err := self.RestClient.Request("getrts", reqData)
	if err != nil {
		return -1, err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return -1, err
		}
		return -1, errors.New(errmsg)
	}

	value, err := GetBodyValue(bodyBytes, "progress")
	if err != nil {
		return -1, err
	}

	return value.(float64), nil
}

/* logout 처리
 */
func (self *BPClient) Logout() *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{}

	success, _, bodyBytes, err := self.RestClient.Request("logout", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* Run 할 테스트 modelname 설정
 */
func (self *BPClient) SetNormalTest(modelname string) *errors.Error {
	if len(modelname) == 0 {
		return errors.New("invalid argument")
	}

	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"template": modelname,
	}

	success, _, bodyBytes, err := self.RestClient.Request("setnormaltest", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* test 설정 변경
 */
func (self *BPClient) ModifyNormalTest(componentId, elementId string, value interface{}) *errors.Error {
	if len(componentId) == 0 || len(elementId) == 0 {
		return errors.New("invalid argument")
	}

	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"componentId": componentId,
		"elementId":   elementId,
		"value":       value,
	}

	success, _, bodyBytes, err := self.RestClient.Request("modifynormaltest", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* 테스트 설정 저장
 */
func (self *BPClient) SaveNormalTest(modelname string) *errors.Error {
	if len(modelname) == 0 {
		return errors.New("invalid argument")
	}

	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"name": modelname,
	}

	success, _, bodyBytes, err := self.RestClient.Request("savenormaltest", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* normal 테스트 설정값 가져옴
 * Client.Response 에 저장됨
 */
func (self *BPClient) ViewNormalTest() *errors.Error {
	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{}

	success, _, bodyBytes, err := self.RestClient.Request("viewnormaltest", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* ViewNormalTest 후에 response body에서 component id, name 축출
 */
func (self *BPClient) GetComponentNames(modelname string) (map[string]string, *errors.Error) {
	if self.RestClient.Rest == nil || self.RestClient.Rest.Response == nil {
		return nil, errors.New("Client, Client.Response is nil")
	}

	err := self.SetNormalTest(modelname)
	if err != nil {
		return nil, err
	}

	err = self.ViewNormalTest()
	if err != nil {
		return nil, err
	}

	bodyUjson, err := utils.NewUjson(self.RestClient.Rest.Response.Body())
	if err != nil {
		return nil, err
	}

	componentList := make(map[string]string)

	for componentId, body := range bodyUjson.Data {
		componentBody, ok := body.(map[string]interface{})
		if !ok {
			continue
		}

		name, ok := componentBody["name"]
		if !ok {
			continue
		}

		nameStr, ok := name.(string)
		if !ok {
			continue
		}

		componentList[componentId] = nameStr
	}

	return componentList, nil
}

/* rfc2544 테스트명 지정
 */
func (self *BPClient) SetRFC(modelname string) *errors.Error {
	if len(modelname) == 0 {
		return errors.New("invalid argument")
	}

	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	reqData := map[Void]Void{
		"template": modelname,
	}

	success, _, bodyBytes, err := self.RestClient.Request("setrfc", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* 결과 pdf 파일 다운로드
 */
func (self *BPClient) ExportTestResult(modelname string, savepath string) *errors.Error {
	if len(modelname) == 0 {
		return errors.New("invalid argument")
	}

	if self.RestClient.Rest == nil {
		return errors.New("bp client doesn't initialize")
	}

	runid, ok := self.RunId[modelname]
	if !ok {
		return errors.New(fmt.Sprintf("'%s' invalid running modelname", modelname))
	}

	reqData := map[Void]Void{
		"runid":        runid,
		"downloadpath": savepath,
	}

	success, _, bodyBytes, err := self.RestClient.Request("exporttestresult", reqData)
	if err != nil {
		return err
	}

	if !success {
		errmsg, err := GetBodyError(bodyBytes)
		if err != nil {
			return err
		}
		return errors.New(errmsg)
	}

	return nil
}

/* 테스트 종료
 */
func (self *BPClient) Close() {
	if self.Done != nil {
		close(self.Done)
		self.Done = nil

		self.Stoptest([]string{})

		time.Sleep(time.Second * 1)

		self.Unreserveports()

		self.Logout()
	}
}

func GetBodyError(body []byte) (string, *errors.Error) {
	value, err := GetBodyValue(body, "error")
	if err != nil {
		return "", err
	}

	if f, ok := value.(float64); ok {
		return strconv.FormatFloat(f, 'f', -1, 64), nil
	} else {
		return fmt.Sprintf("%v", value), nil
	}
}

func GetBodyResult(body []byte) (int, *errors.Error) {
	value, err := GetBodyValue(body, "result")
	if err != nil {
		return RESULT_UNKNOWN, err
	}

	resultMsg := fmt.Sprintf("%v", value)
	if f, ok := value.(float64); ok {
		resultMsg = strconv.FormatFloat(f, 'f', -1, 64)
	}

	if strings.Contains(strings.ToLower(resultMsg), "passed") {
		return RESULT_PASSED, nil
	} else if strings.Contains(strings.ToLower(resultMsg), "failed") {
		return RESULT_FAILED, nil
	} else if strings.Contains(strings.ToLower(resultMsg), "canceled") {
		return RESULT_CANCELED, nil
	} else if strings.Contains(strings.ToLower(resultMsg), "completed") {
		return RESULT_COMPLETED, nil
	} else if strings.Contains(strings.ToLower(resultMsg), "incomplete") {
		return RESULT_INCOMPLETE, nil
	} else if strings.Contains(strings.ToLower(resultMsg), "error") {
		return RESULT_ERROR, nil
	}

	return RESULT_UNKNOWN, errors.New(fmt.Sprintf("'%s' is invalid result string", resultMsg))
}

/* response body에서 key 메시지 찾음
 */
func GetBodyValue(body []byte, key string) (interface{}, *errors.Error) {
	ujson, err := utils.NewUjson(body)
	if err != nil {
		return nil, err
	}

	value, err := ujson.Get(key)
	if err != nil {
		return nil, err
	}

	return value, nil
}
