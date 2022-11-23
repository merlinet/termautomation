package config

import (
	"discovery/constdef"
	"discovery/errors"
	"discovery/fmt"
	"discovery/utils"
	"os"
	"strings"

	"github.com/go-ini/ini"
)

func Version() {
	fmt.Printf("%s %s(Rec. version:%.0f), Multihost command record and replay\n\n", constdef.PRODUCT_NAME, constdef.PRODUCT_VERSION, constdef.RECORD_VERSION)
}

func ReadInstallConf() (string, string, *errors.Error) {
	confPath := fmt.Sprintf("%s/.discovery.conf", os.Getenv("HOME"))
	if _, goerr := os.Stat(confPath); goerr != nil {
		confPath = constdef.INSTALL_ROOT_CONF
	}

	conf, goerr := ini.Load(confPath)
	if goerr != nil {
		return "", "", errors.New(fmt.Sprintf("%s", goerr))
	}

	discoveryRootDir := conf.Section("DEFAULT").Key("DISCOVERY_ROOT").String()
	contentsRootDir := conf.Section("DEFAULT").Key("CONTENTS_ROOT").String()

	if len(discoveryRootDir) == 0 || len(contentsRootDir) == 0 {
		return "", "", errors.New("Invalid discovery install conf")
	}

	if discoveryRootDir[0] != '/' || contentsRootDir[0] != '/' {
		return "", "", errors.New("Invalid discovery install conf")
	}

	return discoveryRootDir, contentsRootDir, nil
}

/* discovery
 */
func GetDiscoveryRoot() (string, *errors.Error) {
	discoveryRootDir, _, err := ReadInstallConf()
	if err != nil {
		return "", err
	}

	return discoveryRootDir, nil
}

func GetDiscoveryBinDir() (string, *errors.Error) {
	dir, err := GetDiscoveryRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/bin", dir), nil
}

func GetDiscoveryEtcDir() (string, *errors.Error) {
	dir, err := GetDiscoveryRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/etc", dir), nil
}

func GetDiscoveryEtcReportserverConfPath() (string, *errors.Error) {
	dir, err := GetDiscoveryEtcDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/report_server.ini", dir), nil
}

func GetDiscoveryEtcCmdControlConfPath() (string, *errors.Error) {
	dir, err := GetDiscoveryEtcDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/cmd_control.ini", dir), nil
}

func GetDiscoveryEtcPromptReStr() (string, *errors.Error) {
	dir, err := GetDiscoveryEtcDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/promptre.conf", dir), nil
}

/* contents
 */
func GetContentsRoot() (string, *errors.Error) {
	_, contentsRootDir, err := ReadInstallConf()
	if err != nil {
		return "", err
	}

	return contentsRootDir, nil
}

func GetContentsDir() (string, *errors.Error) {
	dir, err := GetContentsRoot()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/contents", dir), nil
}

func GetContentsRecordDir(category []string) (string, *errors.Error) {
	dir, err := GetContentsDir()
	if err != nil {
		return "", err
	}

	recordDir := ""
	if len(category) > 0 {
		recordDir = fmt.Sprintf("%s/%s", dir, strings.Join(category, "/"))
	} else {
		recordDir = dir
	}

	return recordDir, nil
}

func GetContentsRecordPrefix(name string, category []string) (string, *errors.Error) {
	recordDir, err := GetContentsRecordDir(category)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s", recordDir, name), nil
}

func GetContentsRecordPath(name string, category []string) (string, *errors.Error) {
	pathPrefix, err := GetContentsRecordPrefix(name, category)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.record", pathPrefix), nil
}

func GetContentsCheckerPath(name string, category []string) (string, *errors.Error) {
	pathPrefix, err := GetContentsRecordPrefix(name, category)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s.checker", pathPrefix), nil
}

func GetContentsEnvDir() (string, *errors.Error) {
	dir, err := GetContentsRoot()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/env", dir), nil
}

func GetContentsEnvFilePath(envCate []string, envName string) (string, *errors.Error) {
	dir, err := GetContentsEnvDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/%s.ini", dir, utils.Rid(envName, envCate)), nil
}

func GetContentsVarDir() (string, *errors.Error) {
	dir, err := GetContentsRoot()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/var", dir), nil
}

func GetContentsResultsDir() (string, *errors.Error) {
	dir, err := GetContentsRoot()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/results", dir), nil
}

func GetContentsSetsDir() (string, *errors.Error) {
	dir, err := GetContentsRoot()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/sets", dir), nil
}

func GetContentsReplaySetFilePath(setname string) (string, *errors.Error) {
	dir, err := GetContentsSetsDir()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s.set", dir, setname), nil
}

func GetLoadPath(loadfile string, RecordCategory []string) (string, *errors.Error) {
	path := ""

	if strings.HasPrefix(strings.ToLower(loadfile), "var:") ||
		strings.HasPrefix(strings.ToLower(loadfile), "env:") ||
		strings.HasPrefix(strings.ToLower(loadfile), "etc:") {

		tmp := strings.SplitN(loadfile, ":", 2)
		if len(tmp) != 2 {
			return "", errors.New("invalid load file path")
		}

		pathPrefix := strings.ToLower(strings.TrimSpace(tmp[0]))
		dir := ""

		switch pathPrefix {
		case "var":
			d, err := GetContentsVarDir()
			if err != nil {
				return "", err
			}
			dir = d

		case "env":
			d, err := GetContentsEnvDir()
			if err != nil {
				return "", err
			}
			dir = d

		case "etc":
			/* etc는 discovery/etc 디렉토리
			 */
			d, err := GetDiscoveryEtcDir()
			if err != nil {
				return "", err
			}
			dir = d

		default:
			return "", errors.New("Invalid load path")
		}

		path = fmt.Sprintf("%s/%s", dir, strings.TrimSpace(tmp[1]))
	} else {
		dir, err := GetContentsDir()
		if err != nil {
			return "", err
		}

		path = fmt.Sprintf("%s/%s", dir, strings.Join(append(RecordCategory, loadfile), "/"))
	}

	return path, nil
}
