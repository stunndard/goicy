package playlist

import (
	"encoding/json"
)

type PlaylistControl struct {
	Reload bool `default:"false"`
	Random bool `default:"false"`
}

type PlaylistContainer struct {
	Playlist Playlist `json:"playlist"`
	Sessions []string `json:"sessions"`
}

// Currently cannot handle mixed public/private downloads
type Playlist struct {
	Name   string         `json:"name"`
	Tracks []Track        `json:"tracks"`
	DlCfg  DownloadConfig `json:"downloadConfig,omitempty"`
}

type DownloadConfig struct {
	Private  bool   `json:"private" default:"true"`
	Endpoint string `json:"endpoint,omitempty"`
	Bucket   string `json:"bucket,omitempty"`
}

type Track struct {
	Title       string `json:"title"`
	Artist      string `json:"artist"`
	Description string `json:"description"`
	Url         string `json:"url"`
	ObjectPath  string `json:"objectPath,omitempty"`
	FilePath    string `json:"filePath,omitempty"`
}

func (pc *PlaylistContainer) PlaylistFromJson(b []byte) {
	json.Unmarshal(b, &pc.Playlist)
}

func (pc *PlaylistContainer) FromJson(b []byte) {
	json.Unmarshal(b, &pc)
}

func (pc *PlaylistContainer) UpdateTrackFilePath(str string, i int) {
	pc.Playlist.Tracks[i].FilePath = str
}

// Appends Filedownloader session to PlaylistContainer
func (pc *PlaylistContainer) AppendFileSession(session string) {
	pc.Sessions = append(pc.Sessions, session)
}

// Returns length of internal Playlist
// Usage:
// if pc.PlaylistLength() > 0 {
// return pc.Playlist.Tracks[0]
// }
func (pc *PlaylistContainer) PlaylistLength() int {
	return len(pc.Playlist.Tracks)

}

// Usage:
// jsonData := []byte(`
//  {
//    "name": "test-plist",
//    "Tracks": [
//      {
//        "title": "unsaved-changes",
//        "artist": "after life",
//        "description": "cool track",
//        "url": "https://oz-tf.nyc3.digitaloceanspaces.com/audio/unsaved-changes.mp3",
//        "filepath": "unsaved-changes.mp3"
//      }
//    ]
//  }`)
// pc := PlaylistContainer
// pc.ToJson(jsonData)
// pc.Playlist.Name
