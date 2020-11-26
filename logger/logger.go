package logger

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/util"
)

const (
	LogError = iota - 1
	LogInfo
	LogDebug
)

func File(s string, level int) {
	var f *os.File
	var err error
	if level > config.Cfg.LogLevel {
		return
	}
	if util.FileExists(config.Cfg.LogFile) {
		f, err = os.OpenFile(config.Cfg.LogFile, os.O_APPEND|os.O_WRONLY, 0666)
		if err != nil {
			return
		}
	} else {
		f, err = os.OpenFile(config.Cfg.LogFile, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			return
		}
	}
	lvl := ""
	switch level {
	case LogError:
		lvl = "ERROR"
	case LogInfo:
		lvl = "INFO "
	case LogDebug:
		lvl = "DEBUG"
	}
	date := time.Now().Format("2006-01-02 15:04:05")
	n, err := f.WriteString("[" + date + "] " + lvl + " " + s + "\r\n")
	if err != nil {
		fmt.Println(n)
		fmt.Println(err)
	}
	//noinspection GoUnhandledErrorResult
	f.Close()
}

func Term(s string, level int) {
	if level > config.Cfg.LogLevel {
		return
	}
	fmt.Print("\r" + strings.Repeat(" ", 79) + "\r" + s)
}

func TermLn(s string, level int) {
	if level > config.Cfg.LogLevel {
		return
	}
	fmt.Println("\r" + strings.Repeat(" ", 79) + "\r" + s)
}

// Logs both to the terminal and the log file.
// Puts ln at the end of the logged string
func Log(s string, level int) {
	TermLn(s, level)
	File(s, level)
}
