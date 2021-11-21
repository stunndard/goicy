package playlist

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/util"
	"github.com/davecgh/go-spew/spew"
)

var playlist []string
var idx int
var np string
var nowPlaying Track

var plc PlaylistContainer

func First() string {
	if plc.PlaylistLength() > 0 {
		return plc.Playlist.Tracks[0].FilePath
	} else {
		return ""
	}
}

func Next(pc PlaylistControl) string {
	if idx > plc.PlaylistLength()-1 {
		idx = 0
	}

	nowPlaying = plc.Playlist.Tracks[idx]

	if pc.Reload {
		LoadJSON()
	}

	for (nowPlaying == plc.Playlist.Tracks[idx]) && (plc.PlaylistLength() > 1) {
		if !config.Cfg.PlayRandom {
			idx = idx + 1
			if idx > plc.PlaylistLength()-1 {
				idx = 0
			} else {
				idx = rand.Intn(plc.PlaylistLength())
			}
		}
	}

	return plc.Playlist.Tracks[idx].FilePath
}

func Load() error {
	// if ok := util.FileExists(config.Cfg.Playlist); !ok {
	// 	return errors.New("Playlist file doesn't exist")
	// }

	content, err := ioutil.ReadFile(config.Cfg.Playlist)
	if err != nil {
		return err
	}

	LoadJSON()

	spew.Dump(plc.Playlist)

	playlist = strings.Split(string(content), "\n")
	i := 0
	for i < len(playlist) {
		playlist[i] = strings.Replace(playlist[i], "\r", "", -1)
		if err != nil {
			return err
		}

		if ok := util.FileExists(playlist[i]); !ok && !strings.HasPrefix(playlist[i], "http") {
			playlist = append(playlist[:i], playlist[i+1:]...)

			continue
		}
		i += 1
	}
	if len(playlist) < 1 {
		return errors.New("Error: all files in the playlist do not exist")
	}
	return nil
}

// Loads json playlist file. Creates a dir configured by `--session-dir` which defaults to `tmp`
func LoadJSON() error {
	if ok := util.FileExists(config.Cfg.Playlist); !ok {
		return errors.New("Playlist file doesn't exist")
	}

	jsonFile, err := os.Open(config.Cfg.Playlist)

	if err != nil {
		fmt.Println("error opening json file")
		return err
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	plc.PlaylistFromJson(byteValue)

	fd := NewDownloader(plc.Playlist.DlCfg)

	plc.AppendFileSession(fd.SessionPath)

	for i, track := range plc.Playlist.Tracks {
		dlf, err := fd.Download(track)
		if err != nil {
			return err
		}

		plc.UpdateTrackFilePath(dlf, i)
	}

	return err
}
