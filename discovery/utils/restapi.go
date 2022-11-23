package utils

import (
	"crypto/tls"
	"discovery/errors"
	"fmt"
	"gopkg.in/resty.v1"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type REST struct {
	Protocol string // HTTP OR HTTPS
	Host     string
	Port     int
	Client   *resty.Client
	Response *resty.Response
}

func NewREST(protocol, host string, port int) (*REST, *errors.Error) {
	if len(host) == 0 {
		return nil, errors.New("invalid host name")
	}

	client := resty.New()
	client.SetRedirectPolicy(resty.FlexibleRedirectPolicy(15))

	switch strings.ToLower(protocol) {
	case "http":
	case "https":
		tlsConfig := &tls.Config{
			InsecureSkipVerify: true,
		}
		client.SetTLSClientConfig(tlsConfig)
	default:
		return nil, errors.New("invalid restful api protocol")
	}

	rest := REST{
		Protocol: protocol,
		Host:     host,
		Port:     port,
		Client:   client,
	}

	return &rest, nil
}

func (self *REST) Url(path string) string {
	rawUrl := ""
	if self.Port != 0 {
		rawUrl = fmt.Sprintf("%s://%s:%d%s", self.Protocol, self.Host, self.Port, path)
	} else {
		rawUrl = fmt.Sprintf("%s://%s%s", self.Protocol, self.Host, path)
	}

	/* encoding query value
	 */
	parsedUrl, err := url.Parse(rawUrl)
	if err != nil {
		fmt.Println("ERR:", err)
		return rawUrl
	}
	parsedUrl.RawQuery = parsedUrl.Query().Encode()

	return parsedUrl.String()
}

func (self *REST) Request(method, url string, headers map[string]string, body string, downloadpath string) *errors.Error {
	if len(method) == 0 || len(url) == 0 {
		return errors.New("invalid arguments")
	}

	switch strings.ToLower(method) {
	case "get":
		return self.Get(url, headers)
	case "put":
		return self.Put(url, headers, body)
	case "post":
		return self.Post(url, headers, body)
	case "delete":
		return self.Delete(url, headers)
	case "patch":
		return self.Patch(url, headers, body)
	case "download":
		return self.Download(url, headers, downloadpath)
	default:
		return errors.New(fmt.Sprintf("'%s' invalid http method", method))
	}

	return nil
}

func (self *REST) Get(urlPath string, headers map[string]string) *errors.Error {
	if len(urlPath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		Get(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	return nil
}

// XXX : 확인 필요, 테스트 안됨
func (self *REST) Put(urlPath string, headers map[string]string, body string) *errors.Error {
	if len(urlPath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		SetBody(body).
		Put(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	return nil
}

func (self *REST) Post(urlPath string, headers map[string]string, body string) *errors.Error {
	if len(urlPath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		SetBody(body).
		Post(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	return nil
}

func (self *REST) Delete(urlPath string, headers map[string]string) *errors.Error {
	if len(urlPath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		Delete(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	return nil
}

func (self *REST) Patch(urlPath string, headers map[string]string, body string) *errors.Error {
	if len(urlPath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		SetBody(body).
		Patch(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	return nil
}

func (self *REST) Download(urlPath string, headers map[string]string, downloadpath string) *errors.Error {
	if len(urlPath) == 0 || len(downloadpath) == 0 {
		return errors.New("invalid argument")
	}

	if self.Client == nil {
		return errors.New("Client is nil")
	}

	resp, oserr := self.Client.R().
		SetHeaders(headers).
		SetDoNotParseResponse(true). // io.Reader 살려두려고 옵션 추가
		Get(self.Url(urlPath))
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	self.Response = resp
	rawbody := resp.RawBody()
	defer rawbody.Close()

	// 파일 save는 status code가 성공일때 수행
	if !self.Response.IsSuccess() {
		return errors.New(fmt.Sprintf("failed to download request, %s", self.Response.Status()))
	}

	err := save(rawbody, downloadpath)
	if err != nil {
		return err
	}

	return nil
}

func save(reader io.ReadCloser, path string) *errors.Error {
	dir := filepath.Dir(path)
	if _, oserr := os.Stat(dir); oserr != nil {
		if os.IsNotExist(oserr) {
			os.MkdirAll(dir, 0755)
		} else {
			return errors.New(fmt.Sprintf("%s", oserr))
		}
	}

	out, oserr := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}
	defer out.Close()

	_, oserr = io.Copy(out, reader)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *REST) Cookies() []*http.Cookie {
	if self.Response != nil {
		return self.Response.Cookies()
	}
	return []*http.Cookie{}
}
