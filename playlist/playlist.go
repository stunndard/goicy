package playlist

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/bgroupe/goicy/config"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/util"
	"github.com/davecgh/go-spew/spew"
)

var playlist []string
var idx int
var np string
var nowPlaying Track

var plc PlaylistContainer

const (
	basePath  = "/tmp/goicy"
	writeMode = 0700
)

func FirstOld() string {
	if len(playlist) > 0 {
		return playlist[0]
	} else {
		return ""
	}
}

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

func NextOld() string {
	//save_idx;

	// get_next_file := pl.Strings[idx];
	if idx > len(playlist)-1 {
		idx = 0
	}
	np = playlist[idx]
	// use current session
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
	return playlist[idx]
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

func downloadFile(fileUrl string, sessionPath string) (string, error) {
	r, err := http.Get(fileUrl)

	if err != nil {
		return "whoops", err
	}
	if r.StatusCode != 200 {
		logger.Log("File not found on remote", 1)
	}
	defer r.Body.Close()

	filePath := path.Base(r.Request.URL.String())

	fullPath := fmt.Sprintf("%s/%s", sessionPath, filePath)

	if _, err := os.Stat(sessionPath); os.IsNotExist(err) {
		os.MkdirAll(sessionPath, writeMode)
	}

	outputFile, err := os.Create(fullPath)

	if err != nil {
		return "file failed to load", err
	}

	defer outputFile.Close()

	_, err = io.Copy(outputFile, r.Body)

	spew.Dump(fullPath)

	return fullPath, err
}

func createBasePathSession() string {
	t := int32(time.Now().Unix())
	return fmt.Sprintf("%s/%v", basePath, t)
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
	bp := createBasePathSession()
	plc.AppendFileSession(bp)

	for i, track := range plc.Playlist.Tracks {
		dlf, err := downloadFile(track.Url, bp)
		if err != nil {
			return err
		}

		plc.UpdateTrackFilePath(dlf, i)
	}

	return err
}
