package utils

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func ParseRid(rid string) (string, []string, *errors.Error) {
	re, oserr := regexp.Compile("[/,;:]+")
	if oserr != nil {
		return "", []string{}, errors.New(fmt.Sprintf("%s", oserr))
	}

	tmpArr := re.Split(rid, -1)
	if len(tmpArr) == 0 {
		return "", []string{}, errors.New("Invalid rid")
	}

	/* rid 문자열 검사
	 */
	for _, ll := range tmpArr {
		if len(ll) == 0 {
			return "", []string{}, errors.New("Invalid rid string")
		}

		if ll[0] == '.' {
			return "", []string{}, errors.New("Invalid rid string")
		}
	}

	name := tmpArr[len(tmpArr)-1]
	cate := tmpArr[:len(tmpArr)-1]

	return name, cate, nil
}

func Rid(name string, category []string) string {
	return strings.Join(append(category, name), "/")
}

func ParseEnvId(rid string) (string, []string, *errors.Error) {
	re, oserr := regexp.Compile("[/,;:]+")
	if oserr != nil {
		return "", []string{}, errors.New(fmt.Sprintf("%s", oserr))
	}

	tmpArr := re.Split(rid, -1)
	if len(tmpArr) == 0 {
		return "", []string{}, errors.New("Invalid envid")
	}

	/* rid 문자열 검사
	 */
	for _, ll := range tmpArr {
		if len(ll) > 0 && ll[0] == '.' {
			return "", []string{}, errors.New("Invalid envid")
		}
	}

	name := tmpArr[len(tmpArr)-1]
	cate := tmpArr[:len(tmpArr)-1]

	return name, cate, nil
}

func PrintResult(resultcode uint8) {
	switch resultcode {
	case constdef.SUCCESS:
		fmt.Printf(" %s-> SUCCESS%s", constdef.ANSI_GREEN_BOLD, constdef.ANSI_END)
	case constdef.FAIL:
		fmt.Printf(" %s-> FAIL%s", constdef.ANSI_RED_BOLD, constdef.ANSI_END)
	default:
		fmt.Printf(" %s-> N/A%s", constdef.ANSI_RED_BOLD, constdef.ANSI_END)
	}
}

func GetQString(arg string) (string, *errors.Error) {
	re, oserr := regexp.Compile(`^\s*"(.*)"\s*$`)
	if oserr != nil {
		return "", errors.New(fmt.Sprintf("%s", oserr))
	}

	resultSlice := re.FindAllStringSubmatch(arg, -1)
	if len(resultSlice) == 0 {
		return "", errors.New("Invalid quoted string, " + arg)
	}

	resultSlice0 := resultSlice[0]

	if len(resultSlice0) != 2 {
		return "", errors.New("Invalid quoted string, " + arg)
	}

	return resultSlice0[1], nil
}

/* record2 호환
 */
func PrintSmile(resultCode *uint8, verbose bool) {
	smileSuccess := ""
	smileFail := ""
	smileNA := ""

	if verbose {
		smileSuccess = fmt.Sprintf(" -> %sSUCCESS%s\n", constdef.ANSI_GREEN_BOLD, constdef.ANSI_END)
		smileFail = fmt.Sprintf(" -> %sFAIL%s\n", constdef.ANSI_RED_BOLD, constdef.ANSI_END)
		smileNA = fmt.Sprintf(" -> %sN/A%s\n", constdef.ANSI_GRAY_BOLD, constdef.ANSI_END)
	} else {
		smileSuccess = fmt.Sprintf("%s☻ %s ", constdef.ANSI_YELLOW_BOLD, constdef.ANSI_END)
		smileFail = fmt.Sprintf("%s☹ %s ", constdef.ANSI_RED_BOLD, constdef.ANSI_END)
		smileNA = fmt.Sprintf("%s☻ %s ", constdef.ANSI_GRAY_BOLD, constdef.ANSI_END)
	}
	switch *resultCode {
	case constdef.CHECKER_SUCCESS:
		fmt.Printf(smileSuccess)
	case constdef.CHECKER_FAIL:
		fmt.Printf(smileFail)
	case constdef.CHECKER_NA:
		fmt.Printf(smileNA)
	default:
	}
}

func GetExecString(arg string) (string, *errors.Error) {
	re, oserr := regexp.Compile("^\\s*`(.*)`\\s*$")
	if oserr != nil {
		return "", errors.New(fmt.Sprintf("%s", oserr))
	}

	resultSlice := re.FindAllStringSubmatch(arg, -1)
	if len(resultSlice) == 0 {
		return "", errors.New("Invalid quoted string, " + arg)
	}

	resultSlice0 := resultSlice[0]

	if len(resultSlice0) != 2 {
		return "", errors.New("Invalid quoted string, " + arg)
	}

	return resultSlice0[1], nil
}

func GetStringArray(arg string) ([]string, *errors.Error) {
	if len(arg) < 2 || arg[0] != '{' || arg[len(arg)-1] != '}' {
		return []string{}, errors.New("Invalid argument")
	}
	argStr := arg[1 : len(arg)-1]
	if len(argStr) == 0 {
		return []string{}, errors.New("Invalid array string, " + arg)
	}

	strArr := []string{}
	for _, rawStr := range strings.Split(argStr, ",") {
		str, err := GetQString(rawStr)
		if err != nil {
			return []string{}, err
		}

		strArr = append(strArr, str)
	}

	return strArr, nil
}

func GetSingleVariable(input string) (string, string, int, *errors.Error) {
	re, goerr := regexp.Compile(`^\s*(\$|\$#|@)<\s*([\w:]+)\s*>(?:\[\s*(.+)\s*\])*\s*$`)
	if goerr != nil {
		return "", "", -1, errors.New(fmt.Sprintf("%s", goerr))
	}

	matchedArr := re.FindAllStringSubmatch(input, -1)
	if len(matchedArr) == 0 {
		return "", "", -1, errors.New(fmt.Sprintf("%s is not a variable", input))
	}

	item := matchedArr[0]
	if len(item) != 4 {
		return "", "", -1, errors.New("Invalid variable string")
	}

	vartype := item[1]
	varname := item[2]
	arridxstr := item[3]

	arridx := -1
	if len(arridxstr) > 0 {
		if vartype == "$#" || vartype == "@" {
			return "", "", -1, errors.New("invalid variable name syntax")
		}

		n, goerr1 := strconv.ParseUint(arridxstr, 10, 64)
		if goerr1 != nil {
			return "", "", -1, errors.New(fmt.Sprintf("%s", goerr1))
		}
		arridx = int(n)
	}

	return vartype, varname, arridx, nil
}

func GetSingleArrayVariable(input string) (string, *errors.Error) {
	vartype, varname, _, err := GetSingleVariable(input)
	if err != nil {
		return "", err
	}

	if vartype != "@" {
		return "", errors.New("Invalid array variable string")
	}

	return varname, nil
}

/* check rcmd의 exit_code, output_string, output_line_count 파싱 함수
 */
func GetCheckLegacyCmdStr(input string) (string, string, *errors.Error) {
	if len(input) == 0 {
		return "", "", errors.New("Invalid input string")
	}

	re, goerr := regexp.Compile(`(\w+)(?:\[(\d+)\])?`)
	if goerr != nil {
		return "", "", errors.New(fmt.Sprintf("%s", goerr))
	}

	resultSlice := re.FindAllStringSubmatch(input, -1)
	if len(resultSlice) == 0 {
		return "", "", errors.New("Invalid input string")
	}

	resultSlice0 := resultSlice[0]

	if len(resultSlice0) != 3 {
		return "", "", errors.New("Invalid input string")
	}

	cmdStr := resultSlice0[1]
	arrIdxStr := resultSlice0[2]

	return cmdStr, arrIdxStr, nil
}

/* os.Args 에서 record version(-rv) 옵션값 확인
 * default 값은 constdef.RECORD_VERSION
 */
func GetRecordVersionArg() (string, *errors.Error) {
	for i, str := range os.Args {
		if str == "-rv" {
			if i < len(os.Args)-1 {
				args := os.Args[:i]
				i++
				ver := os.Args[i]
				i++
				args = append(args, os.Args[i:]...)
				os.Args = args

				return ver, nil
			} else {
				return "", errors.New("invalid -rv arguments")
			}
		}
	}

	return fmt.Sprintf("%.0f", constdef.RECORD_VERSION), nil
}

/* 따움표 붙은 문자열에서 따옴표 제거
 */
func Unquote(text string) string {
	s := strings.TrimSpace(text)
	if len(s) < 2 {
		return text
	}

	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return strings.ReplaceAll(s[1:len(s)-1], "\\'", "'")
	} else if s[0] == '"' && s[len(s)-1] == '"' {
		return strings.ReplaceAll(s[1:len(s)-1], "\\\"", "\"")
	} else if s[0] == '`' && s[len(s)-1] == '`' {
		return strings.ReplaceAll(s[1:len(s)-1], "\\`", "`")
	} else {
		return text
	}
}

/* text 안 문자를 확인하고 따옴표 붙임
 */
func Quote(text string) string {
	if strings.IndexByte(text, '"') == -1 {
		text = `"` + text + `"`
	} else if strings.IndexByte(text, '\'') == -1 {
		text = `'` + text + `'`
	} else if strings.IndexByte(text, '`') == -1 {
		text = "`" + text + "`"
	} else {
		text = `"` + strings.Replace(text, `"`, `\"`, -1) + `"`
	}

	return text
}

/* record 2 의 $<variable>[3] -> variable[3] 문자열로 변환
 * $#<variable> -> len(variable)
 * @<variable> -> variable
 */
func ConvVari2to3(text string) string {
	varRe, goerr := regexp.Compile(`(?:\$|\$#|@)<([^>]*)>`)
	if goerr != nil {
		return text
	}

	output := text
	matchedArr := varRe.FindAllStringSubmatch(output, -1)

	/* matched 변수 문자열 array
	 */
	for _, item := range matchedArr {
		if len(item) != 2 {
			return output
		}

		from := item[0]
		to := item[1]

		if strings.HasPrefix(from, "$#") {
			to = fmt.Sprintf("len(%s)", to)
		}

		output = strings.Replace(output, from, to, -1)
	}

	return output
}

/* address=seoul 값을 address을 key, seoul 을 value로 잘라서 return
 */
func ParseKeyValueString(data string, sep string) (string, string, *errors.Error) {
	if len(data) == 0 {
		return "", "", errors.New("invalid arguments")
	}

	arr := strings.SplitN(data, sep, 2)
	if len(arr) == 0 {
		return "", "", errors.New("invalid arguments")
	} else if len(arr) == 1 {
		return strings.TrimSpace(arr[0]), "", nil
	} else {
		return strings.TrimSpace(arr[0]), strings.TrimSpace(arr[1]), nil
	}
}
