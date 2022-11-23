package record3

import (
	"discovery/config"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"github.com/alecthomas/repr"
)

const EnvironmentRcmdStr = "environment"

type Environment struct {
	Name    string `@"environment"`
	EnvId   string `@STRING`
	EnvHash string `@STRING`
	Env     *config.Env
}

func NewEnvironment(text string) (*Environment, *errors.Error) {
	target, err := NewStruct(text, &Environment{})
	if err != nil {
		return nil, err
	}
	return target.(*Environment), nil
}

func NewEnvironment2(envid string) (*Environment, *errors.Error) {
	if len(envid) == 0 {
		return nil, errors.New("Invalid arguments")
	}

	env, err := config.NewEnv(envid)
	if err != nil {
		return nil, err
	}

	environment := Environment{
		Name:    EnvironmentRcmdStr,
		EnvId:   utils.Quote(envid),
		EnvHash: utils.Quote(env.EnvHash),
		Env:     env,
	}

	return &environment, nil
}

func (self *Environment) ToString() string {
	return fmt.Sprintf("%s %s %s", self.Name, self.EnvId, self.EnvHash)
}

func (self *Environment) Prepare(context *ReplayerContext) *errors.Error {
	if self.Env == nil {
		envid := utils.Unquote(self.EnvId)
		if context != nil && len(context.ForceEnvId) > 0 {
			forceEnvName, forceEnvCate, err := utils.ParseEnvId(context.ForceEnvId)
			if err != nil {
				return err
			}

			envName, envCate, err := utils.ParseRid(envid)
			if err != nil {
				return err
			}

			/* force envid 가 /single_route 이면 기존 카테고리는 보존하고 env name만 변경
			 */
			if context.ForceEnvId[0] == '/' ||
				context.ForceEnvId[0] == ',' ||
				context.ForceEnvId[0] == ';' ||
				context.ForceEnvId[0] == ':' {

				if len(forceEnvName) > 0 {
					envName = forceEnvName
				} else {
					return errors.New(fmt.Sprintf("%s, invalid force env id", context.ForceEnvId))
				}
				/* force envid 가 rss_single_route/ 이면 기존 env name는 보존하고 env category만 변경
				 */
			} else if context.ForceEnvId[len(context.ForceEnvId)-1] == '/' ||
				context.ForceEnvId[len(context.ForceEnvId)-1] == ',' ||
				context.ForceEnvId[len(context.ForceEnvId)-1] == ';' ||
				context.ForceEnvId[len(context.ForceEnvId)-1] == ':' {

				if len(forceEnvCate) > 0 {
					envCate = forceEnvCate
				} else {
					return errors.New(fmt.Sprintf("%s, invalid force env id", context.ForceEnvId))
				}
			} else {
				envCate = forceEnvCate
				envName = forceEnvName
			}

			envid = utils.Rid(envName, envCate)
		}

		env, err := config.NewEnv(envid)
		if err != nil {
			return err
		}
		self.Env = env
	}

	if context != nil && context.NoEnvHashCheck {
	} else {
		if utils.Unquote(self.EnvHash) != self.Env.EnvHash {
			return errors.New("environment hash string doesn't matched").AddMsg(self.ToString())
		}
	}

	return nil
}

func (self *Environment) Do(context *ReplayerContext) (Void, *errors.Error) {
	if context == nil {
		return nil, errors.New("invalid arguments").AddMsg(self.ToString())
	}

	if context.Env != nil {
		return nil, errors.New(fmt.Sprintf("%s environment already loaded", utils.Unquote(self.EnvId))).AddMsg(self.ToString())
	}

	if self.Env == nil {
		return nil, errors.New(fmt.Sprintf("%s environment is not init.", utils.Unquote(self.EnvId))).AddMsg(self.ToString())
	}

	context.Env = self.Env

	/* env node 정보 varMap에 load
	 */
	envLoad, err := NewLoad(fmt.Sprintf(`load ini "%s"`, fmt.Sprintf("env:%s.ini", utils.Rid(self.Env.EnvName, self.Env.EnvCategory))))
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	_, err = envLoad.Do(context)
	if err != nil {
		return nil, err.AddMsg(self.ToString())
	}

	return nil, nil
}

func (self *Environment) GetName() string {
	return self.Name
}

func (self *Environment) Dump() {
	repr.Println(self)
}
