package config

import (
	"bufio"
	"discovery/errors"
	"discovery/fmt"
	"os"
	"regexp"
	"strings"
)

/* regexp string을 멀티로 설정
 */
type PromptRe struct {
	Str string
	Re  *regexp.Regexp
}

func NewPromptRe(reStr string) (*PromptRe, *errors.Error) {
	if len(reStr) == 0 {
		return nil, errors.New("invalid reStr arguments")
	}

	re, oserr := regexp.Compile(reStr)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	p := PromptRe{
		Str: reStr,
		Re:  re,
	}

	return &p, nil
}

type PromptRegex struct {
	List []*PromptRe
}

func NewPromptRegex() (*PromptRegex, *errors.Error) {
	path, err := GetDiscoveryEtcPromptReStr()
	if err != nil {
		return nil, err
	}

	fp, oserr := os.Open(path)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}
	defer fp.Close()

	promptregex := PromptRegex{}
	reader := bufio.NewReader(fp)
	for {
		data, _, oserr := reader.ReadLine()
		if oserr != nil {
			break
		}

		promptstr := strings.TrimSpace(string(data))
		if len(promptstr) == 0 {
			continue
		}

		pre, err := NewPromptRe(promptstr)
		if err != nil {
			return nil, err
		}

		promptregex.List = append(promptregex.List, pre)
	}

	return &promptregex, nil
}

func (self *PromptRegex) MatchPrompt(promptstr string) (bool, string, *errors.Error) {
	for _, promptre := range self.List {
		if promptre.Re.MatchString(promptstr) {
			return true, promptre.Str, nil
		}
	}

	return false, "", errors.New("not matched")
}
