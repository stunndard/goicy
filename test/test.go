package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	"github.com/bgroupe/goicy/playlist"
	"github.com/bgroupe/goicy/util"
	"github.com/davecgh/go-spew/spew"
)

const jsonPath = "/Users/caseyguerrero/persona/goicy/playlist.json"

func main() {

	var plc playlist.PlaylistContainer

	if ok := util.FileExists(jsonPath); !ok {
		fmt.Println("Playlist doesn't exist")
		// return errors.New("Playlist file doesn't exist")
	}

	jsonFile, err := os.Open(jsonPath)

	if err != nil {
		fmt.Println("error opening json file")
	}
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	plc.PlaylistFromJson(byteValue)

	spew.Dump(plc)
	filePath := path.Base(plc.Playlist.Tracks[0].Url)
	spew.Println(filePath)

	plc.UpdateTrackFilePath(filePath, 0)
	spew.Dump(plc)

}
