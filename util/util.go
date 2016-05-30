package util

import (
	"os"
	"strings"
)

func FileExists(name string) bool {
	finfo, err := os.Stat(name)
	if err != nil {
		// no such file or dir
		return false
	}
	return !finfo.IsDir()
}

func Basename(s string) string {
	n := strings.LastIndexByte(s, '.')
	if n >= 0 {
		return s[:n]
	}
	return s
}
