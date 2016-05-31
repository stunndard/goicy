package util

import (
	"os"
	"strings"
)

type FileError struct {
	Msg string
}

func (e *FileError) Error() string {
	return e.Msg
}

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
