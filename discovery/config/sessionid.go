package config

import (
	"fmt"
	"os"
)

func GetSessionId(prefix string) string {
	if len(prefix) > 0 {
		return fmt.Sprintf("%s_%d", prefix, os.Getpid())
	} else {
		return fmt.Sprintf("SID%d", os.Getpid())
	}
}
