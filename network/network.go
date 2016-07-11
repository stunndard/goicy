package network

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/logger"
	"net"
	"strconv"
	"time"
)

var Connected bool = false
var csock net.Conn

func Connect(host string, port int) (net.Conn, error) {

	h := host + ":" + strconv.Itoa(int(port))
	sock, err := net.Dial("tcp", h)
	if err != nil {
		Connected = false
	}
	return sock, err
}

func Send(sock net.Conn, buf []byte) error {
	n, err := sock.Write(buf)
	if (err != nil) || (n < 1) {
		Connected = false
	}
	return err
}

func Recv(sock net.Conn) ([]byte, error) {
	var buf []byte = make([]byte, 1024)

	n, err := sock.Read(buf)
	//fmt.Println(n, err, string(buf), len(buf))
	if err != nil {
		logger.Log(err.Error(), logger.LOG_ERROR)
		return nil, err
	}

	return buf[0:n], err
}

func Close(sock net.Conn) {
	Connected = false
	sock.Close()
}

func ConnectServer(host string, port int, br float64, sr, ch int) (net.Conn, error) {
	var sock net.Conn

	if Connected {
		return csock, nil
	}

	if config.Cfg.ServerType == "shoutcast" {
		port++
	}
	logger.Log("Connecting to "+config.Cfg.ServerType+" at "+host+":"+strconv.Itoa(port)+"...", logger.LOG_DEBUG)
	sock, err := Connect(host, port)

	if err != nil {
		Connected = false
		return sock, err
	}

	//fmt.Println("connected ok")
	time.Sleep(time.Second)

	headers := ""
	bitrate := 0
	samplerate := 0
	channels := 0

	if config.Cfg.StreamType == "file" {
		bitrate = int(br)
		samplerate = sr
		channels = ch
	} else {
		bitrate = config.Cfg.StreamBitrate / 1000
		samplerate = config.Cfg.StreamSamplerate
		channels = config.Cfg.StreamChannels
	}

	contenttype := ""
	if config.Cfg.StreamFormat == "mpeg" {
		contenttype = "audio/mpeg"
	} else {
		contenttype = "audio/aacp"
	}

	if config.Cfg.ServerType == "shoutcast" {
		if err := Send(sock, []byte(config.Cfg.Password+"\r\n")); err != nil {
			logger.Log("Error sending password", logger.LOG_ERROR)
			Connected = false
			return sock, err
		}

		time.Sleep(time.Second)

		resp, err := Recv(sock)
		if err != nil {
			logger.Log("Error receiving ShoutCast response", logger.LOG_ERROR)
			Connected = false
			return sock, err
		}
		//fmt.Println(string(resp[0:3]))
		if string(resp[0:3]) != "OK2" {
			logger.Log("Shoutcast password rejected: "+string(resp), logger.LOG_ERROR)
			Connected = false
			return sock, err
		}
		//fmt.Println("password accepted")
		headers = "content-type:" + contenttype + "\r\n" +
			"icy-name:" + config.Cfg.StreamName + "\r\n" +
			"icy-genre:" + config.Cfg.StreamGenre + "\r\n" +
			"icy-url:" + config.Cfg.StreamURL + "\r\n" +
			"icy-pub:0\r\n" +
			fmt.Sprintf("icy-br:%d\r\n\r\n", bitrate)
	} else {
		headers = "SOURCE /" + config.Cfg.Mount + " HTTP/1.0\r\n" +
			"Content-Type: " + contenttype + "\r\n" +
			"Authorization: Basic " + base64.StdEncoding.EncodeToString([]byte("source:"+config.Cfg.Password)) + "\r\n" +
			"User-Agent: goicy/" + config.Version + "\r\n" +
			"ice-name: " + config.Cfg.StreamName + "\r\n" +
			"ice-public: 0\r\n" +
			"ice-url: " + config.Cfg.StreamURL + "\r\n" +
			"ice-genre: " + config.Cfg.StreamGenre + "\r\n" +
			"ice-description: " + config.Cfg.StreamDescription + "\r\n" +
			"ice-audio-info: bitrate=" + strconv.Itoa(bitrate) +
			";channels=" + strconv.Itoa(channels) +
			";samplerate=" + strconv.Itoa(samplerate) + "\r\n" +
			"\r\n"
	}

	if err := Send(sock, []byte(headers)); err != nil {
		logger.Log("Error sending headers", logger.LOG_ERROR)
		Connected = false
		return sock, err
	}

	if config.Cfg.ServerType == "icecast" {
		time.Sleep(time.Second)
		resp, err := Recv(sock)
		if err != nil {
			Connected = false
			return sock, err
		}
		if string(resp[9:12]) != "200" {
			Connected = false
			return sock, errors.New("Invalid Icecast response: " + string(resp))
		}
	}

	logger.Log("Server connect successful", logger.LOG_INFO)
	Connected = true
	csock = sock

	return sock, nil
}
