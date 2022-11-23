package config

import (
	"crypto/sha256"
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"encoding/hex"
	"reflect"
	"sort"
	"strings"

	"github.com/go-ini/ini"
)

var NODE_TABLE = map[string]NodeInterface{
	"linux":  &NodeLinux{},
	"ssh":    &NodeSSH{},
	"telnet": &NodeTelnet{},
	"rest":   &NodeRest{},
	"docker": &NodeDockerContainer{},
	"cisco":  &NodeCisco{},
}

type NodeInterface interface {
	MapTo(section *ini.Section) *errors.Error
	GetConnectCommand() (string, *errors.Error)
	GetLoginRcmdList(func(bool, string)) *errors.Error

	GetString(field string) (string, *errors.Error)
	GetInt(field string) (int, *errors.Error)
	GetArrInt(field string) ([]int, *errors.Error)

	CanBash() bool
	NewEmpty() NodeInterface
	Dump(depth string)
}

/* struct 의 변수이름으로 값 찾아 리턴
 */
func getFieldValue(value reflect.Value, field string) (reflect.Value, *errors.Error) {
	v := reflect.Indirect(value).FieldByName(field)
	if !v.IsValid() {
		return value, errors.New(fmt.Sprintf("'%s' is invalid field name", field))
	}

	return v, nil
}

// SSH
type NodeSSH struct {
	Ip           string `ini:"ip"`
	Port         int    `ini:"port"`
	Username     string `ini:"username"`
	Password     string `ini:"password"`
	CharacterSet string `ini:"character_set"`
	Eol          string `ini:"eol"`
}

func (self *NodeSSH) MapTo(section *ini.Section) *errors.Error {
	self.Eol = constdef.DEFAULT_EOL
	self.CharacterSet = constdef.DEFAULT_CHARACTER_SET

	oserr := section.MapTo(self)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *NodeSSH) GetConnectCommand() (string, *errors.Error) {
	if len(self.Username) == 0 || len(self.Ip) == 0 || self.Port <= 0 {
		return "", errors.New("ssh config. has invalid arguments")
	}

	command := fmt.Sprintf("/usr/bin/ssh %s@%s -p %d", self.Username, self.Ip, self.Port)
	return command, nil
}

func (self *NodeSSH) GetLoginRcmdList(callback func(needExpectFlag bool, sendStr string)) *errors.Error {
	if len(self.Password) > 0 {
		callback(true, self.Password)
	}
	return nil
}

func (self *NodeSSH) GetString(field string) (string, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (self *NodeSSH) GetInt(field string) (int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return int(-1), err
	}

	return int(v.Int()), nil
}

func (self *NodeSSH) GetArrInt(field string) ([]int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return []int{}, err
	}

	return v.Interface().([]int), nil
}

func (self *NodeSSH) CanBash() bool {
	return true
}

func (self *NodeSSH) NewEmpty() NodeInterface {
	return &NodeSSH{}
}

func (self *NodeSSH) Dump(depth string) {
	fmt.Println(depth + "SSH:")
	fmt.Println(depth+" Ip:", self.Ip)
	fmt.Println(depth+" Port:", self.Port)
	fmt.Println(depth+" Username:", self.Username)
	fmt.Println(depth+" Password:", self.Password)
	fmt.Println(depth+" CharacterSet:", self.CharacterSet)
	fmt.Println(depth+" Eol:", self.Eol)
}

// Telnet
type NodeTelnet struct {
	Ip           string `ini:"ip"`
	Port         int    `ini:"port"`
	Username     string `ini:"username"`
	Password     string `ini:"password"`
	CharacterSet string `ini:"character_set"`
	Eol          string `ini:"eol"`
}

func (self *NodeTelnet) MapTo(section *ini.Section) *errors.Error {
	self.Eol = constdef.DEFAULT_EOL
	self.CharacterSet = constdef.DEFAULT_CHARACTER_SET

	oserr := section.MapTo(self)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *NodeTelnet) GetConnectCommand() (string, *errors.Error) {
	if len(self.Ip) == 0 || self.Port <= 0 {
		return "", errors.New("telnet config. has invalid arguments")
	}

	command := fmt.Sprintf("/usr/bin/telnet %s %d", self.Ip, self.Port)
	return command, nil
}

func (self *NodeTelnet) GetLoginRcmdList(callback func(needExpectFlag bool, sendStr string)) *errors.Error {
	if len(self.Username) > 0 {
		callback(true, self.Username)

		if len(self.Password) > 0 {
			callback(true, self.Password)
		}
	}
	return nil
}

func (self *NodeTelnet) GetString(field string) (string, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (self *NodeTelnet) GetInt(field string) (int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return int(-1), err
	}

	return int(v.Int()), nil
}

func (self *NodeTelnet) GetArrInt(field string) ([]int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return []int{}, err
	}

	return v.Interface().([]int), nil
}

func (self *NodeTelnet) CanBash() bool {
	return true
}

func (self *NodeTelnet) NewEmpty() NodeInterface {
	return &NodeTelnet{}
}

func (self *NodeTelnet) Dump(depth string) {
	fmt.Println(depth + "Telnet:")
	fmt.Println(depth+" Ip:", self.Ip)
	fmt.Println(depth+" Port:", self.Port)
	fmt.Println(depth+" Username:", self.Username)
	fmt.Println(depth+" Password:", self.Password)
	fmt.Println(depth+" CharacterSet:", self.CharacterSet)
	fmt.Println(depth+" Eol:", self.Eol)
}

// Rest
type NodeRest struct {
	Ip               string `ini:"ip"`
	RestPort         int    `ini:"rest_port"`
	RestProtocol     string `ini:"rest_protocol"` // http, https
	RestApiPath      string `ini:"rest_api"`
	RestCharacterSet string `ini:"rest_character_set"`
	RestEol          string `ini:"rest_eol"`
}

func (self *NodeRest) MapTo(section *ini.Section) *errors.Error {
	self.RestEol = constdef.DEFAULT_EOL
	self.RestCharacterSet = constdef.DEFAULT_CHARACTER_SET

	oserr := section.MapTo(self)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *NodeRest) GetConnectCommand() (string, *errors.Error) {
	return "", errors.New("NodeRest doesn't support connection command")
}

func (self *NodeRest) GetLoginRcmdList(callback func(needExpectFlag bool, sendStr string)) *errors.Error {
	return errors.New("NodeRest doesn't support connection login rcmd list")
}

func (self *NodeRest) GetString(field string) (string, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (self *NodeRest) GetInt(field string) (int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return int(-1), err
	}

	return int(v.Int()), nil
}

func (self *NodeRest) GetArrInt(field string) ([]int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return []int{}, err
	}

	return v.Interface().([]int), nil
}

func (self *NodeRest) CanBash() bool {
	return false
}

func (self *NodeRest) NewEmpty() NodeInterface {
	return &NodeRest{}
}

func (self *NodeRest) Dump(depth string) {
	fmt.Println(depth + "Rest:")
	fmt.Println(depth+" Ip:", self.Ip)
	fmt.Println(depth+" RestPort:", self.RestPort)
	fmt.Println(depth+" RestProtocol:", self.RestProtocol)
	fmt.Println(depth+" RestApiPath:", self.RestApiPath)
	fmt.Println(depth+" RestCharacterSet:", self.RestCharacterSet)
	fmt.Println(depth+" RestEol:", self.RestEol)
}

// Linux
type NodeLinux struct {
	NodeSSH
}

func (self *NodeLinux) NewEmpty() NodeInterface {
	return &NodeLinux{}
}

func (self *NodeLinux) Dump(depth string) {
	fmt.Println(depth + "NodeLinux:")
	self.NodeSSH.Dump(depth + " ")
}

// Cisco
type NodeCisco struct {
	Ip           string `ini:"ip"`
	Port         int    `ini:"port"`
	Password     string `ini:"password"`
	CharacterSet string `ini:"character_set"`
	Eol          string `ini:"eol"`
}

func (self *NodeCisco) MapTo(section *ini.Section) *errors.Error {
	self.Eol = constdef.EOL_CR
	self.CharacterSet = constdef.DEFAULT_CHARACTER_SET

	oserr := section.MapTo(self)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *NodeCisco) GetConnectCommand() (string, *errors.Error) {
	if len(self.Ip) == 0 || self.Port <= 0 {
		return "", errors.New("telnet config. has invalid arguments")
	}

	command := fmt.Sprintf("/usr/bin/telnet %s %d", self.Ip, self.Port)
	return command, nil
}

func (self *NodeCisco) GetLoginRcmdList(callback func(needExpectFlag bool, sendStr string)) *errors.Error {
	if len(self.Password) > 0 {
		callback(true, self.Password)
	}
	return nil
}

func (self *NodeCisco) GetString(field string) (string, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (self *NodeCisco) GetInt(field string) (int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return int(-1), err
	}

	return int(v.Int()), nil
}

func (self *NodeCisco) GetArrInt(field string) ([]int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return []int{}, err
	}

	return v.Interface().([]int), nil
}

func (self *NodeCisco) CanBash() bool {
	return false
}

func (self *NodeCisco) NewEmpty() NodeInterface {
	return &NodeCisco{}
}

func (self *NodeCisco) Dump(depth string) {
	fmt.Println(depth + "NodeCisco:")
	fmt.Println(depth+" Ip:", self.Ip)
	fmt.Println(depth+" Port:", self.Port)
	fmt.Println(depth+" Password:", self.Password)
	fmt.Println(depth+" CharacterSet:", self.CharacterSet)
	fmt.Println(depth+" Eol:", self.Eol)
}

// Docker Container
// 더 구상 필요
type NodeDockerContainer struct {
	NodeSSH
	ContainerName string `ini:"docker_container"`
}

func (self *NodeDockerContainer) MapTo(section *ini.Section) *errors.Error {
	err := self.NodeSSH.MapTo(section)
	if err != nil {
		return err
	}

	oserr := section.MapTo(self)
	if oserr != nil {
		return errors.New(fmt.Sprintf("%s", oserr))
	}

	return nil
}

func (self *NodeDockerContainer) GetLoginRcmdList(callback func(needExpectFlag bool, sendStr string)) *errors.Error {
	err := self.NodeSSH.GetLoginRcmdList(callback)
	if err != nil {
		return err
	}

	if len(self.ContainerName) > 0 {
		callback(true, fmt.Sprintf("docker exec -it \"%s\" /bin/bash", self.ContainerName))
	}

	return nil
}

func (self *NodeDockerContainer) GetString(field string) (string, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return "", err
	}

	return v.String(), nil
}

func (self *NodeDockerContainer) GetInt(field string) (int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return int(-1), err
	}

	return int(v.Int()), nil
}

func (self *NodeDockerContainer) GetArrInt(field string) ([]int, *errors.Error) {
	v, err := getFieldValue(reflect.ValueOf(self), field)
	if err != nil {
		return []int{}, err
	}

	return v.Interface().([]int), nil
}

func (self *NodeDockerContainer) NewEmpty() NodeInterface {
	return &NodeDockerContainer{}
}

func (self *NodeDockerContainer) Dump(depth string) {
	fmt.Println(depth + "NodeDockerContainer:")
	self.NodeSSH.Dump(depth + " ")
	fmt.Println(depth+" ContainerName:", self.ContainerName)
}

/* Node
 */
type Node struct {
	Name     string
	NodeType string `ini:"type"`
	NodeInfo NodeInterface
}

func (self *Node) Dump(depth string) {
	fmt.Println(depth+"Name:", self.Name)
	fmt.Println(depth+"NodeType:", self.NodeType)
	self.NodeInfo.Dump(depth)
}

type Env struct {
	EnvConfPath string
	EnvCategory []string
	EnvName     string
	Desc        string
	NodeList    map[string]*Node
	EnvHash     string
}

func NewEnv(envid string) (*Env, *errors.Error) {
	envName, envCate, err := utils.ParseRid(envid)
	if err != nil {
		return nil, err
	}

	path, err := GetContentsEnvFilePath(envCate, envName)
	if err != nil {
		return nil, err
	}

	env := Env{
		EnvConfPath: path,
		EnvCategory: envCate,
		EnvName:     envName,
		NodeList:    make(map[string]*Node),
	}

	err = env.load()
	if err != nil {
		return nil, err
	}

	env.GenHash()

	return &env, nil
}

func (self *Env) load() *errors.Error {
	conf, goerr := ini.LoadSources(ini.LoadOptions{IgnoreInlineComment: true}, self.EnvConfPath)
	if goerr != nil {
		return errors.New(fmt.Sprintf("%s", goerr))
	}

	secNames := conf.SectionStrings()
	for _, name := range secNames {
		if name == "DEFAULT" {
			continue
		}

		if name == "COMMON" {
			self.Desc = conf.Section("COMMON").Key("description").String()
			continue
		}

		node := &Node{
			Name: name,
		}

		iniSection := conf.Section(node.Name)
		goerr = iniSection.MapTo(node)
		if goerr != nil {
			return errors.New(fmt.Sprintf("%s", goerr))
		}

		node.NodeType = strings.ToLower(node.NodeType)

		nodeinterface, ok := NODE_TABLE[node.NodeType]
		if !ok {
			return errors.New(fmt.Sprintf("'%s' is invalid node type", node.NodeType))
		}

		nodeinfo := nodeinterface.NewEmpty()

		err := nodeinfo.MapTo(iniSection)
		if err != nil {
			return err
		}
		node.NodeInfo = nodeinfo

		self.NodeList[node.Name] = node
	}

	return nil
}

func (self *Env) GetNode(nodename string) *Node {
	node, ok := self.NodeList[nodename]
	if ok {
		return node
	} else {
		return nil
	}
}

func (self *Env) GetNodeInfo(nodename string) (NodeInterface, *errors.Error) {
	node := self.GetNode(nodename)
	if node == nil {
		return nil, errors.New(fmt.Sprintf("'%s' node is not defined", nodename))
	}

	return node.NodeInfo, nil
}

/* environment hash 값 생성
 * hash 생성 string은 envid, node name, node type
 * 으로 이것이 같으면 같은 테스트 구성으로 간주함
 */
func (self *Env) GenHash() *errors.Error {
	/* map range 시 순서가 바뀔 수 있어서, sorting 함
	 */
	var sortedNodeList []string
	for key, _ := range self.NodeList {
		sortedNodeList = append(sortedNodeList, key)
	}
	sort.Strings(sortedNodeList)

	arr := make([]string, 0)
	for _, key := range sortedNodeList {
		node, ok := self.NodeList[key]
		if ok {
			arr = append(arr, fmt.Sprintf("%s,%s", strings.ToLower(node.Name), strings.ToLower(node.NodeType)))
		}
	}

	rawStr := strings.Join(arr, " ")

	hash := sha256.New()
	hash.Write([]byte(rawStr))
	hashStr := hex.EncodeToString(hash.Sum(nil))

	self.EnvHash = hashStr

	return nil
}

func (self *Env) Dump() {
	fmt.Println("EnvConfPath:", self.EnvConfPath)
	fmt.Println("EnvCategory:", self.EnvCategory)
	fmt.Println("EnvName:", self.EnvName)
	fmt.Println("Desc:", self.Desc)

	for _, node := range self.NodeList {
		node.Dump(" ")
	}

	fmt.Println("EnvHash:", self.EnvHash)
}
