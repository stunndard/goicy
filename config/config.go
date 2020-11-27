package config

import (
	"github.com/go-ini/ini"
)

type Config struct {
	StreamType        string `ini:"streamtype"`
	StreamFormat      string `ini:"format"`
	StreamReencode    bool   `ini:"reencode"`
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
	FFMPEGTimeout     int    `ini:"timeout"`
	PidFile           string
	FFMPEGPath        string
}

const Version = "0.3"

var Cfg Config

func LoadConfig(filename string) error {

	iniFile, err := ini.Load(filename)
	if err != nil {
		return err
	}

	Cfg.ServerType = iniFile.Section("server").Key("server").Value()
	Cfg.Host = iniFile.Section("server").Key("host").Value()
	Cfg.Port, _ = iniFile.Section("server").Key("port").Int()
	Cfg.Mount = iniFile.Section("server").Key("mount").Value()
	Cfg.ConnAttempts, _ = iniFile.Section("server").Key("connectionattempts").Int()
	Cfg.Password = iniFile.Section("server").Key("password").Value()

	Cfg.StreamType = iniFile.Section("stream").Key("streamtype").Value()
	Cfg.StreamFormat = iniFile.Section("stream").Key("format").Value()
	Cfg.StreamReencode, _ = iniFile.Section("ffmpeg").Key("reencode").Bool()
	Cfg.StreamBitrate, _ = iniFile.Section("ffmpeg").Key("bitrate").Int()
	Cfg.StreamChannels, _ = iniFile.Section("ffmpeg").Key("channels").Int()
	Cfg.StreamSamplerate, _ = iniFile.Section("ffmpeg").Key("samplerate").Int()
	Cfg.StreamAACProfile = iniFile.Section("ffmpeg").Key("aacprofile").Value()
	Cfg.FFMPEGPath = iniFile.Section("ffmpeg").Key("ffmpeg").Value()
	Cfg.FFMPEGTimeout, _ = iniFile.Section("ffmpeg").Key("timeout").Int()

	Cfg.StreamName = iniFile.Section("stream").Key("name").Value()
	Cfg.StreamDescription = iniFile.Section("stream").Key("description").Value()
	Cfg.StreamURL = iniFile.Section("stream").Key("url").Value()
	Cfg.StreamGenre = iniFile.Section("stream").Key("genre").Value()
	Cfg.StreamPublic, _ = iniFile.Section("stream").Key("public").Bool()

	Cfg.PlaylistType = iniFile.Section("playlist").Key("playlisttype").Value()
	Cfg.Playlist = iniFile.Section("playlist").Key("playlist").Value()
	Cfg.PlayRandom, _ = iniFile.Section("playlist").Key("playrandom").Bool()

	Cfg.BufferSize, _ = iniFile.Section("misc").Key("buffersize").Int()
	Cfg.BufferSize *= 1000
	Cfg.UpdateMetadata, _ = iniFile.Section("misc").Key("updatemetadata").Bool()
	Cfg.ScriptFile = iniFile.Section("misc").Key("script").Value()
	Cfg.NpFile = iniFile.Section("misc").Key("npfile").Value()
	Cfg.LogFile = iniFile.Section("misc").Key("logfile").Value()
	Cfg.LogLevel, _ = iniFile.Section("misc").Key("loglevel").Int()
	Cfg.IsDaemon, _ = iniFile.Section("misc").Key("daemon").Bool()
	Cfg.PidFile = iniFile.Section("misc").Key("pidfile").Value()

	return nil
}

func init() {
	Cfg.LogLevel = 1
	Cfg.LogFile = "goicy.log"
}
