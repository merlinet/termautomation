package utils

import (
	"bufio"
	"crypto/md5"
	"crypto/sha256"
	"discovery/errors"
	"discovery/fmt"
	"encoding/base64"
	"encoding/hex"
	"io/ioutil"
	"os"
)

func GetFileSha256Hash(filePath string) (string, *errors.Error) {
	fp, goerr := os.Open(filePath)
	if goerr != nil {
		return "", errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	hash := sha256.New()
	reader := bufio.NewReader(fp)
exit:
	for {
		data := make([]byte, 256)
		n, goerr := reader.Read(data)
		if goerr != nil {
			break exit
		}

		hash.Write(data[:n])
	}

	hashStr := hex.EncodeToString(hash.Sum(nil))
	return hashStr, nil
}

func GetFileMd5Hash(filePath string) (string, *errors.Error) {
	fp, goerr := os.Open(filePath)
	if goerr != nil {
		return "", errors.New(fmt.Sprintf("%s", goerr))
	}
	defer fp.Close()

	hash := md5.New()
	reader := bufio.NewReader(fp)
exit:
	for {
		data := make([]byte, 256)
		n, goerr := reader.Read(data)
		if goerr != nil {
			break exit
		}

		hash.Write(data[:n])
	}

	hashStr := hex.EncodeToString(hash.Sum(nil))
	return hashStr, nil
}

func GetFileBase64(filePath string) ([]string, *errors.Error) {
	fp, oserr := os.Open(filePath)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	reader := bufio.NewReader(fp)
	data, oserr := ioutil.ReadAll(reader)
	if oserr != nil {
		return nil, errors.New(fmt.Sprintf("%s", oserr))
	}

	encData := base64.StdEncoding.EncodeToString(data)

	var encDataArr []string

	for {
		if len(encData) > 76 {
			encDataArr = append(encDataArr, encData[:76])
			encData = encData[76:]
		} else {
			encDataArr = append(encDataArr, encData)
			break
		}
	}

	return encDataArr, nil
}
