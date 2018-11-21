package playlist

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"strings"

	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/plugin"
	"github.com/stunndard/goicy/util"
)

var (
	playlist     []string
	idx          int
	np           string
	pluginActive bool
)

func First() (string, string) {
	if len(playlist) > 0 {
		if idx > len(playlist)-1 {
			return "", ""
		}
		if strings.HasPrefix(playlist[idx], "plugin://") {
			err := plugin.Load(playlist[idx])
			if err != nil {
				logger.Log(err.Error(), logger.LOG_ERROR)
				idx =+ 1
				return First()
			}
			file, metadata, err := plugin.First()
			if err != nil {
				logger.Log(err.Error(), logger.LOG_ERROR)
				idx =+ 1
				return First()
			}
			pluginActive = file != "plugin://no.more"
			if pluginActive {
				return file, metadata
			} else {
				idx =+ 1
				return First()
			}
		}
		return playlist[idx], ""
	} else {
		return "", ""
	}
}

func Next() (string, string) {

	if pluginActive {
		file, metadata, err := plugin.Next()
		if err != nil {
			logger.Log(err.Error(), logger.LOG_ERROR)
			return "", ""
		}
		pluginActive = file != "plugin://no.more"
		if pluginActive {
			return file, metadata
		}
	}

	if idx > len(playlist)-1 {
		idx = 0
	}
	np = playlist[idx]
	Load()
	if idx > len(playlist)-1 {
		idx = 0
	}
	for (np == playlist[idx]) && (len(playlist) > 1) {
		if !config.Cfg.PlayRandom {
			idx = idx + 1
			if idx > len(playlist)-1 {
				idx = 0
			}
		} else {
			idx = rand.Intn(len(playlist))
		}
	}
	return playlist[idx], ""
}

func Load() error {
	if ok := util.FileExists(config.Cfg.Playlist); !ok {
		return errors.New("playlist file doesn't exist")
	}

	content, err := ioutil.ReadFile(config.Cfg.Playlist)
	if err != nil {
		return err
	}
	playlist = strings.Split(string(content), "\n")

	i := 0
	for i < len(playlist) {
		playlist[i] = strings.Replace(playlist[i], "\r", "", -1)
		if ok := util.FileExists(playlist[i]); !ok && !strings.HasPrefix(playlist[i], "http") &&
			!strings.HasPrefix(playlist[i], "plugin://") {
			playlist = append(playlist[:i], playlist[i+1:]...)
			continue
		}
		i += 1
	}
	if len(playlist) < 1 {
		return errors.New("error: all files in the playlist do not exist")
	}

	return nil
}
