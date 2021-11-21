package main

import (
	"fmt"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/playlist"
	"github.com/bgroupe/goicy/stream"
	"github.com/bgroupe/goicy/util"

	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {

	fmt.Println("=====================================================================")
	fmt.Println(" goicy v" + config.Version + " -- A hz reincarnate rewritten in Go")
	fmt.Println(" AAC/AACplus/AACplusV2 & MP1/MP2/MP3 Icecast/Shoutcast source client")
	fmt.Println(" Copyright (C) 2006-2016 Roman Butusov <reaxis at mail dot ru>")
	fmt.Println("=====================================================================")
	fmt.Println()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		stream.Abort = true
		logger.Log("Aborted by user/SIGTERM", logger.LOG_INFO)
	}()

	if len(os.Args) < 2 {
		fmt.Println("Usage: goicy <inifile>")
		return
	}
	inifile := string(os.Args[1])

	//inifile := "d:\\work\\src\\Go\\src\\github.com\\stunndard\\goicy\\tests\\goicy.ini"

	logger.TermLn("Loading config...", logger.LOG_DEBUG)
	err := config.LoadConfig(inifile)
	if err != nil {
		logger.TermLn(err.Error(), logger.LOG_ERROR)
		return
	}
	logger.File("---------------------------", logger.LOG_INFO)
	logger.File("goicy v"+config.Version+" started", logger.LOG_INFO)
	logger.Log("Loaded config file: "+inifile, logger.LOG_INFO)

	defer logger.Log("goicy exiting", logger.LOG_INFO)

	if err := playlist.LoadJSON(); err != nil {
		logger.Log("Cannot load playlist file", logger.LOG_ERROR)
		logger.Log(err.Error(), logger.LOG_ERROR)
		return
	}
	// default values
	playlistCtrl := playlist.PlaylistControl{}

	retries := 0
	filename := playlist.First()
	for {
		var err error
		if config.Cfg.StreamType == "file" {
			err = stream.StreamFile(filename)
		} else {
			err = stream.StreamFFMPEG(filename)
		}

		if err != nil {
			// if aborted break immediately
			if stream.Abort {
				break
			}
			retries++
			logger.Log("Error streaming: "+err.Error(), logger.LOG_ERROR)

			if retries == config.Cfg.ConnAttempts {
				logger.Log("No more retries", logger.LOG_INFO)
				break
			}
			// if that was a file error
			switch err.(type) {
			case *util.FileError:
				filename = playlist.Next(playlistCtrl)
			default:

			}

			logger.Log("Retrying in 10 sec...", logger.LOG_INFO)
			for i := 0; i < 10; i++ {
				time.Sleep(time.Second * 1)
				if stream.Abort {
					break
				}
			}
			if stream.Abort {
				break
			}
			continue
		}
		retries = 0

		filename = playlist.Next(playlistCtrl)
	}
}
