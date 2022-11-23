/* general restful api client 처리
 */
package record3

import (
	goctx "context"
	"discovery/errors"
	"discovery/utils"
	"encoding/json"
	"fmt"
	goversion "github.com/hashicorp/go-version"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
)

type RestClient struct {
	Rest       *utils.REST
	SetCookies []*http.Cookie
	Ctx        goctx.Context // aips _csrf 값 유지
	ApiPath    string
	RequestStr string
}

func NewRestClient(protocol, host string, port int, apipath string) (*RestClient, *errors.Error) {
	if len(host) == 0 || len(apipath) == 0 {
		return nil, errors.New("invalid argument")
	}

	rest, err := utils.NewREST(protocol, host, port)
	if err != nil {
		return nil, err
	}

	client := RestClient{
		Rest:    rest,
		Ctx:     goctx.Background(),
		ApiPath: apipath,
	}

	return &client, nil
}

func (self *RestClient) Request(command string, referenceData map[Void]Void) (bool, int, []byte, *errors.Error) {
	if len(command) == 0 || referenceData == nil {
		return false, 0, []byte{}, errors.New("invalid arguments")
	}

	if self.Rest == nil {
		return false, 0, []byte{}, errors.New("Rest client doesn't init.")
	}

	jsonpath := fmt.Sprintf("%s/%s.json", self.ApiPath, command)

	ujsondata, err := utils.NewUjsonWithPath(jsonpath)
	if err != nil {
		return false, 0, []byte{}, err
	}

	method, err := ujsondata.GetString("method")
	if err != nil {
		return false, 0, []byte{}, err
	}

	/* restapi version 비교
	 */
	version, _ := ujsondata.GetString("version")
	err = self.CompareApiVersion(version)
	if err != nil {
		return false, 0, []byte{}, err
	}

	/* referenceData, referenceJsonData
	 * url encoded: urlEncodedReferenceData, urlEncodedReferenceJsonData
	 */
	tmpRes := ConvVoidToStringMap(referenceData)
	referenceJsonData, ok := tmpRes.(map[string]interface{})
	if !ok {
		return false, 0, []byte{}, errors.New("Failed to ConvVoidToStringMap()")
	}

	urlEncodedReferenceData := make(map[Void]Void)
	for k, v := range referenceData {
		s, ok := v.(string)
		if ok {
			urlEncodedReferenceData[k] = url.QueryEscape(s)
		} else {
			urlEncodedReferenceData[k] = v
		}
	}

	tmpRes = ConvVoidToStringMap(urlEncodedReferenceData)
	urlEncodedReferenceJsonData, ok := tmpRes.(map[string]interface{})
	if !ok {
		return false, 0, []byte{}, errors.New("Failed to ConvVoidToStringMap()")
	}

	urn, err := ujsondata.GetString("urn")
	if err != nil {
		return false, 0, []byte{}, err
	}

	/* urn {DATA} 문자열 변수 치환
	 */
	urn, err = replaceRefString(urn, referenceData)
	if err != nil {
		return false, 0, []byte{}, err
	}

	headers := make(map[string]string)

	/* 쿠치 set-cookie 값 헤더 포함
	 */
	if len(self.SetCookies) > 0 {
		var c string
		for _, cookie := range self.SetCookies {
			if len(c) == 0 {
				c = cookie.String()
			} else {
				c = c + ";" + cookie.String()
			}
		}
		headers["Cookie"] = c
	}

	/* header 값 처리
	 */
	headersData, err := ujsondata.GetData("headers")
	if err == nil {
		replacedHeadersData, err := replaceJsonData(headersData, referenceJsonData)
		if err != nil {
			return false, 0, []byte{}, err
		}

		for k, v := range replacedHeadersData {
			/* {DATA} 문자열 변수 치환
			 */
			keyStr, err := replaceRefString(k, referenceData)
			if err != nil {
				return false, 0, []byte{}, err
			}

			valuetmp := fmt.Sprintf("%v", v)
			if f, ok := v.(float64); ok {
				valuetmp = strconv.FormatFloat(f, 'f', -1, 64)
			}
			valueStr, err := replaceRefString(valuetmp, referenceData)
			if err != nil {
				return false, 0, []byte{}, err
			}
			headers[keyStr] = valueStr
		}
	}

	/* header encoding이  urlencoded 이면 id, password url encoding 수행
	 */
	refData := referenceData
	refJsonData := referenceJsonData

	contentType, ok := headers["Content-Type"]
	if ok && strings.Contains(contentType, "urlencoded") {
		refData = urlEncodedReferenceData
		refJsonData = urlEncodedReferenceJsonData
	}

	payload := ""
	payloadData, err := ujsondata.Get("payload")
	if err == nil {
		var payloadStr string

		switch payloadData.(type) {
		case map[string]interface{}:
			replacedPayloadData, err := replaceJsonData(payloadData.(map[string]interface{}), refJsonData)
			if err != nil {
				return false, 0, []byte{}, err
			}

			jsonBytes, oserr := json.Marshal(replacedPayloadData)
			if oserr != nil {
				return false, 0, []byte{}, errors.New(fmt.Sprintf("%s", oserr))
			}
			// json marshal 하면 \r -> \\r, \n -> \\n 으로 자동 변환하는데..
			// 이거 다시 원복
			payloadStr = strings.ReplaceAll(string(jsonBytes), "\\\\r", "\\r")
			payloadStr = strings.ReplaceAll(payloadStr, "\\\\n", "\\n")
		default:
			if f, ok := payloadData.(float64); ok {
				payloadStr = strconv.FormatFloat(f, 'f', -1, 64)
			} else {
				payloadStr = fmt.Sprintf("%v", payloadData)
			}
		}

		/* payload {DATA} 문자열 변수 치환
		 */
		payloadStr, err = replaceRefString(payloadStr, refData)
		if err != nil {
			return false, 0, []byte{}, err
		}

		payload = payloadStr
	}

	downloadpath := ""
	if strings.ToLower(method) == "download" {
		path, err := ujsondata.GetString("downloadpath")
		if err != nil {
			return false, 0, []byte{}, err
		}
		/* {} 변수 치환
		 */
		path, err = replaceRefString(path, referenceData)
		if err != nil {
			return false, 0, []byte{}, err
		}
		downloadpath = path
	}

	/* request info
	 */
	self.RequestStr = fmt.Sprintf("RESTAPI=%s, %s, headers:%v, payload:%s, downloadpath:%s",
		strings.ToUpper(method), self.Rest.Url(urn), headers, payload, downloadpath)

	err = self.Rest.Request(method, urn, headers, payload, downloadpath)
	if err != nil {
		return false, 0, []byte{}, err
	}

	/* response
	 */
	success := self.Rest.Response.IsSuccess()
	statusCode := self.Rest.Response.StatusCode()
	body := self.Rest.Response.Body()

	/* response에서 set cookie 처리
	 */
	setCookies := self.Rest.Cookies()
	if len(setCookies) > 0 {
		self.SetCookies = setCookies
	}

	return success, statusCode, body, nil
}

func (self *RestClient) CompareApiVersion(targetVersion string) *errors.Error {
	if self.Ctx == nil {
		return errors.New("restclient context is nil")
	}

	var version string
	versionData := self.Ctx.Value("version")

	if versionData == nil {
		// specific version 이 없으면, OK
		return nil
	}

	v, ok := versionData.(string)
	if !ok {
		return errors.New(fmt.Sprintf("'%v', restclient ctx has invalid version data", versionData))
	}

	if len(v) == 0 {
		// specific version 이 없으면, OK
		return nil
	}
	version = v

	if len(targetVersion) == 0 {
		return errors.New(fmt.Sprintf("required restapi version is %s", version))
	}

	target, goerr := goversion.NewVersion(targetVersion)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	constraints, goerr := goversion.NewConstraint(version)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	if constraints.Check(target) == false {
		return errors.New(fmt.Sprintf("required restapi version is %s, but %s", constraints, target))
	}
	return nil
}

/* 문자열의 {변수} 안의 값의 referenceData 값으로 치환
 * "payload": {
 *    "username": "{username}",
 *    "password": "{password}"
 * }
 */
func replaceRefString(msg string, referenceData map[Void]Void) (string, *errors.Error) {
	varRe, goerr := regexp.Compile(`{([A-Za-z_]+[A-Za-z0-9:_]*)}`)
	if goerr != nil {
		return msg, errors.New(fmt.Sprintf("%s", goerr))
	}

	return doReplaceRefString(varRe, msg, referenceData)
}

func doReplaceRefString(varRe *regexp.Regexp, msg string, referenceData map[Void]Void) (string, *errors.Error) {
	if varRe == nil {
		return msg, errors.New("Invalid arguments")
	}

	input := msg
	matchedArr := varRe.FindAllStringSubmatch(input, -1)
	if len(matchedArr) == 0 {
		return input, nil
	}

	/* matched 변수 문자열 array
	 */
	for _, item := range matchedArr {
		if len(item) != 2 {
			return input, errors.New("invalid string replacing format")
		}

		from := item[0]
		text := item[1]

		res, ok := referenceData[text]
		if !ok {
			return input, errors.New(fmt.Sprintf("doReplaceRefString(), '%s' is invalid variable name", text))
		}
		to := fmt.Sprintf("%v", res)
		if f, ok := res.(float64); ok {
			to = strconv.FormatFloat(f, 'f', -1, 64)
		}

		input = strings.Replace(input, from, to, -1)
	}

	return doReplaceRefString(varRe, input, referenceData)
}

/* json map data의 key값을 referenceData 참조하여 replace함
 */
func replaceJsonData(jsonData map[string]interface{},
	referenceData map[string]interface{}) (map[string]interface{}, *errors.Error) {

	for key, value := range jsonData {
		newvalue, ok := referenceData[key]
		if ok {
			jsonData[key] = newvalue.(interface{})
		} else {
			switch value.(type) {
			case map[string]interface{}:
				res, err := replaceJsonData(value.(map[string]interface{}), referenceData)
				if err != nil {
					return nil, err
				}
				jsonData[key] = res
			}
		}
	}

	return jsonData, nil
}
