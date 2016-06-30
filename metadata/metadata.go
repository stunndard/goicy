package metadata

import (
	"encoding/base64"
	"github.com/go-ini/ini"
	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/network"
	"net/url"
	"os/exec"
	"strings"
)

func FormatMetadata(artist, title string) string {
	md := ""
	if artist != "" {
		md = artist + " - " + title
	} else {
		md = title
	}
	if md == "" {
		md = config.Cfg.StreamName
	}
	return md
}

func SendMetadata(metadata string) error {
	logger.Log("Setting metadata: "+metadata, logger.LOG_INFO)
	sock, err := network.Connect(config.Cfg.Host, config.Cfg.Port)
	if err != nil {
		return err
	}

	headers := ""
	if config.Cfg.ServerType == "shoutcast" {
		headers = "GET /admin.cgi?pass=" + url.QueryEscape(config.Cfg.Password) +
			"&mode=updinfo&song=" + strings.Replace(url.QueryEscape(metadata), "+", "%20", -1) + " HTTP/1.0\r\n" +
			"User-Agent: (Mozilla Compatible)\r\n\r\n"
	} else {
		headers = "GET /admin/metadata?mode=updinfo&mount=/" + config.Cfg.Mount +
			"&song=" + strings.Replace(url.QueryEscape(metadata), "+", "%20", -1) + " HTTP/1.0\r\n" +
			"User-Agent: goicy/" + config.Version + "\r\n" +
			"Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("source:"+config.Cfg.Password)) + "\r\n\r\n"
	}
	if err := network.Send(sock, []byte(headers)); err != nil {
		return err
	}
	return nil
}

func GetTagsFFMPEG(filename string) error {
	cmdName := config.Cfg.FFMPEGPath
	cmdArgs := []string{
		"-i", filename,
		"-f", "ffmetadata",
		"-",
	}

	logger.Log("Launching FFMPEG to read tags...", logger.LOG_DEBUG)
	cmd := exec.Command(cmdName, cmdArgs...)

	out, err := cmd.Output()
	if err != nil {
		return err
	}

	ini, err := ini.Load(out)
	if err != nil {
		return err
	}

	section, _ := ini.GetSection("")
	artist := section.Key("artist").Value()
	if artist == "" {
		artist = section.Key("ARTIST").Value()
	}

	title := section.Key("title").Value()
	if title == "" {
		title = section.Key("TITLE").Value()
	}

	logger.Log("Artist: "+artist, logger.LOG_DEBUG)
	logger.Log("Title: "+title, logger.LOG_DEBUG)

	// format metadata
	metadata := FormatMetadata(artist, title)

	// send it
	if err := SendMetadata(metadata); err != nil {
		return err
	}

	return nil
}
