package util

import (
	"os"
	"strings"
)

func StringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func IsSystemInTestMode() bool {
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.") {
			return true
		}
	}
	return false
}
