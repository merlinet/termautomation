package fmt

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
)

var MultiWriterFp io.Writer = nil

var (
	offset  = 0
	webFlag = false
)

func InitPrintWeb(paths []string) error {
	webFlag = true
	fplist := []io.Writer{}

	if len(paths) == 0 {
		return errors.New("invalid arguments")
	}

	for _, path := range paths {
		fp, oserr := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if oserr != nil {
			return oserr
		}
		fplist = append(fplist, fp)
	}

	MultiWriterFp = io.MultiWriter(fplist...)
	return nil
}

func InitPrint(paths []string) error {
	fplist := []io.Writer{os.Stdout}

	for _, path := range paths {
		fp, oserr := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
		if oserr != nil {
			return oserr
		}
		fplist = append(fplist, fp)
	}

	MultiWriterFp = io.MultiWriter(fplist...)
	return nil
}

func Printf(format string, a ...interface{}) (n int, err error) {
	if MultiWriterFp == nil {
		return fmt.Printf(format, a...)
	} else {
		msg := fmt.Sprintf(format, a...)
		n, err = MultiWriterFp.Write([]byte(msg))
		if err == nil && webFlag == true {
			encoded_msg := base64.StdEncoding.EncodeToString([]byte(msg))
			std_msg := fmt.Sprintf("{\"offset\": \"%d\", \"msg\": \"%s\", \"type\": \"printf\"}", offset, encoded_msg)
			_, std_err := fmt.Println(std_msg)
			if std_err != nil {
				return n, std_err
			}
			offset += n
		}
		return n, err
	}
}

func Println(a ...interface{}) (n int, err error) {
	if MultiWriterFp == nil {
		return fmt.Println(a...)
	} else {
		msg := fmt.Sprintln(a...)
		n, err = MultiWriterFp.Write([]byte(msg))
		if err == nil && webFlag == true {
			encoded_msg := base64.StdEncoding.EncodeToString([]byte(msg))
			std_msg := fmt.Sprintf("{\"offset\": \"%d\", \"msg\": \"%s\", \"type\": \"println\"}", offset, encoded_msg)
			_, std_err := fmt.Println(std_msg)
			if std_err != nil {
				return n, std_err
			}
			offset += n
		}
		return n, err
	}
}

func Sprintf(format string, a ...interface{}) string {
	return fmt.Sprintf(format, a...)
}

func Sprintln(a ...interface{}) string {
	return fmt.Sprintln(a...)
}

func Fprintf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, format, a...)
}

func Fprintln(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprintln(w, a...)
}
