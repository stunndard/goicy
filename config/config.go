package config

import (
	"github.com/go-ini/ini"
)

type Config struct {
	StreamType        string `ini:"streamtype"`
	StreamFormat      string `ini:"format"`
	StreamBitrate     int    `ini:"bitrate"`
	StreamChannels    int    `ini:"channels"`
	StreamSamplerate  int    `ini:"samplerate"`
	StreamAACProfile  string `ini:"aacprofile"`
	ServerType        string `ini:"server"`
	Host              string `ini:"host"`
	Port              int    `ini:"port"`
	Mount             string `ini:"mount"`
	ConnAttempts      int    `ini:"connectionattempts"`
	Password          string `ini:"password"`
	BufferSize        int    `ini:"buffersize"`
	Playlist          string `ini:"playlist"`
	PlaylistType      string `ini:"playlistype"`
	NpFile            string `ini:"npfile"`
	LogFile           string `ini:"logfile"`
	ScriptFile        string `ini:"logfile"`
	LogLevel          int    `ini:"loglevel"`
	PlayRandom        bool   `ini:"playrandom"`
	UpdateMetadata    bool   `ini:"updatemetadata"`
	StreamName        string `ini:"name"`
	StreamDescription string `ini:"description"`
	StreamURL         string `ini:"url"`
	StreamGenre       string `ini:"genre"`
	StreamPublic      bool   `ini:"public"`
	IsDaemon          bool   `ini:"daemon"`
	PidFile           string
	FFMPEGPath        string
}

const Version = "0.3"

var Cfg Config

func LoadConfig(filename string) error {

	ini, err := ini.Load(filename)
	if err != nil {
		return err
	}

	Cfg.ServerType = ini.Section("server").Key("server").Value()
	Cfg.Host = ini.Section("server").Key("host").Value()
	Cfg.Port, _ = ini.Section("server").Key("port").Int()
	Cfg.Mount = ini.Section("server").Key("mount").Value()
	Cfg.ConnAttempts, _ = ini.Section("server").Key("connectionattempts").Int()
	Cfg.Password = ini.Section("server").Key("password").Value()

	Cfg.StreamType = ini.Section("stream").Key("streamtype").Value()
	Cfg.StreamFormat = ini.Section("stream").Key("format").Value()
	Cfg.StreamBitrate, _ = ini.Section("ffmpeg").Key("bitrate").Int()
	Cfg.StreamChannels, _ = ini.Section("ffmpeg").Key("channels").Int()
	Cfg.StreamSamplerate, _ = ini.Section("ffmpeg").Key("samplerate").Int()
	Cfg.StreamAACProfile = ini.Section("ffmpeg").Key("aacprofile").Value()
	Cfg.FFMPEGPath = ini.Section("ffmpeg").Key("ffmpeg").Value()

	Cfg.StreamName = ini.Section("stream").Key("name").Value()
	Cfg.StreamDescription = ini.Section("stream").Key("description").Value()
	Cfg.StreamURL = ini.Section("stream").Key("url").Value()
	Cfg.StreamGenre = ini.Section("stream").Key("genre").Value()
	Cfg.StreamPublic, _ = ini.Section("stream").Key("public").Bool()

	Cfg.PlaylistType = ini.Section("playlist").Key("playlisttype").Value()
	Cfg.Playlist = ini.Section("playlist").Key("playlist").Value()
	Cfg.PlayRandom, _ = ini.Section("playlist").Key("playrandom").Bool()

	Cfg.BufferSize, _ = ini.Section("misc").Key("buffersize").Int()
	Cfg.BufferSize *= 1000
	Cfg.UpdateMetadata, _ = ini.Section("misc").Key("updatemetadata").Bool()
	Cfg.ScriptFile = ini.Section("misc").Key("script").Value()
	Cfg.NpFile = ini.Section("misc").Key("npfile").Value()
	Cfg.LogFile = ini.Section("misc").Key("logfile").Value()
	Cfg.LogLevel, _ = ini.Section("misc").Key("loglevel").Int()
	Cfg.IsDaemon, _ = ini.Section("misc").Key("daemon").Bool()
	Cfg.PidFile = ini.Section("misc").Key("pidfile").Value()

	return nil
}

func init() {
	Cfg.LogLevel = 1
	Cfg.LogFile = "goicy.log"
}
