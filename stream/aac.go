package stream

import (
	"errors"
	"github.com/stunndard/goicy/aac"
	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/cuesheet"
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/metadata"
	"github.com/stunndard/goicy/network"
	"github.com/stunndard/goicy/util"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"
)

var totalFramesSent uint64
var totalTimeBegin time.Time
var Abort bool

func StreamAACFile(filename string) error {
	var (
		br                  float64
		spf, sr, frames, ch int
		sock                net.Conn
	)

	cleanUp := func(err error) {
		network.Close(sock)
		//totalFramesSent = 0
	}

	logger.Log("Checking file: "+filename+"...", logger.LOG_INFO)

	if err := aac.GetFileInfo(filename, &br, &spf, &sr, &frames, &ch); err != nil {
		return err
	}

	var err error
	sock, err = network.ConnectServer(config.Cfg.Host, config.Cfg.Port, br, sr, ch)
	if err != nil {
		logger.Log("Cannot connect to server", logger.LOG_ERROR)
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		cleanUp(err)
		return err
	}

	defer f.Close()

	aac.SeekTo1StFrame(*f)

	logger.Log("Streaming file: "+filename+"...", logger.LOG_INFO)

	cuefile := util.Basename(filename) + ".cue"
	if config.Cfg.UpdateMetadata {
		go metadata.GetTagsFFMPEG(filename)
		cuesheet.Load(cuefile)
	}

	logger.TermLn("CTRL-C to stop", logger.LOG_INFO)

	framesSent := 0

	// get number of frames to read in 1 iteration
	framesToRead := (sr / spf) + 1
	timeBegin := time.Now()

	for framesSent < frames {
		sendBegin := time.Now()

		lbuf, err := aac.GetFrames(*f, framesToRead)
		if err != nil {
			logger.Log("Error reading data stream", logger.LOG_ERROR)
			cleanUp(err)
			return err
		}

		if err := network.Send(sock, lbuf); err != nil {
			cleanUp(err)
			logger.Log("Error sending data stream", logger.LOG_ERROR)
			return err
		}

		framesSent = framesSent + framesToRead

		timeElapsed := int(float64((time.Now().Sub(timeBegin)).Seconds()) * 1000)
		timeSent := int(float64(framesSent) * float64(spf) / float64(sr) * 1000)

		bufferSent := 0
		if timeSent > timeElapsed {
			bufferSent = timeSent - timeElapsed
		}

		if config.Cfg.UpdateMetadata {
			cuesheet.Update(uint32(timeElapsed))
		}

		// calculate the send lag
		sendLag := int(float64((time.Now().Sub(sendBegin)).Seconds()) * 1000)

		if timeElapsed > 1500 {
			logger.Term("Frames: "+strconv.Itoa(framesSent)+"/"+strconv.Itoa(frames)+"  Time: "+
				strconv.Itoa(timeElapsed)+"/"+strconv.Itoa(timeSent)+"ms  Buffer: "+
				strconv.Itoa(bufferSent)+"  Bps: "+strconv.Itoa(len(lbuf)), logger.LOG_INFO)
		}

		// regulate sending rate
		timePause := 0
		if bufferSent < (config.Cfg.BufferSize - 100) {
			timePause = 900 - sendLag
		} else {
			if bufferSent > config.Cfg.BufferSize {
				timePause = 1100 - sendLag
			} else {
				timePause = 975 - sendLag
			}
		}

		if Abort {
			err := errors.New("aborted by user")
			cleanUp(err)
			return err
		}

		time.Sleep(time.Duration(time.Millisecond) * time.Duration(timePause))
	}

	// pause to clear up the buffer
	timeBetweenTracks := int(((float64(frames)*float64(spf))/float64(sr))*1000) - int(float64((time.Now().Sub(timeBegin)).Seconds())*1000)
	logger.Log("Pausing for "+strconv.Itoa(timeBetweenTracks)+"ms...", logger.LOG_DEBUG)
	time.Sleep(time.Duration(time.Millisecond) * time.Duration(timeBetweenTracks))

	return nil
}

func StreamAACFFMPEG(filename string) error {
	var (
		sock net.Conn
		res  error
		cmd  *exec.Cmd
	)

	cleanUp := func(err error) {
		logger.Log("Killing ffmpeg..", logger.LOG_DEBUG)
		cmd.Process.Kill()
		network.Close(sock)
		totalFramesSent = 0
		res = err
	}

	var err error
	sock, err = network.ConnectServer(config.Cfg.Host, config.Cfg.Port, 0, 0, 0)
	if err != nil {
		logger.Log("Cannot connect to server", logger.LOG_ERROR)
		return err
	}

	aacprofile := ""

	if config.Cfg.StreamAACProfile == "lc" {
		aacprofile = "aac_low"
	} else if config.Cfg.StreamAACProfile == "he" {
		aacprofile = "aac_he"
	} else {
		aacprofile = "aac_he_v2"
	}

	cmdArgs := []string{
		"-i", filename,
		"-c:a", "libfdk_aac",
		"-profile:a", aacprofile, //"aac_low", //
		"-b:a", strconv.Itoa(config.Cfg.StreamBitrate),
		"-cutoff", "20000",
		"-ar", strconv.Itoa(config.Cfg.StreamSamplerate),
		//"-ac", strconv.Itoa(config.Cfg.StreamChannels),
		"-f", "adts",
		"-",
	}

	logger.Log("Starting ffmpeg: "+config.Cfg.FFMPEGPath, logger.LOG_DEBUG)
	logger.Log("Format         : "+aacprofile, logger.LOG_DEBUG)
	logger.Log("Bitrate        : "+strconv.Itoa(config.Cfg.StreamBitrate), logger.LOG_DEBUG)
	logger.Log("Samplerate     : "+strconv.Itoa(config.Cfg.StreamSamplerate), logger.LOG_DEBUG)

	cmd = exec.Command(config.Cfg.FFMPEGPath, cmdArgs...)

	f, _ := cmd.StdoutPipe()

	//cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		logger.Log("Error starting ffmpeg", logger.LOG_ERROR)
		logger.Log(err.Error(), logger.LOG_ERROR)
		return err
	}

	logger.Log("Streaming file: "+filename+"...", logger.LOG_INFO)

	cuefile := util.Basename(filename) + ".cue"
	if config.Cfg.UpdateMetadata {
		go metadata.GetTagsFFMPEG(filename)
		cuesheet.Load(cuefile)
	}

	logger.TermLn("CTRL-C to stop", logger.LOG_INFO)

	frames := 0
	timeFileBegin := time.Now()

	// get number of frames to read in 1 iteration
	// for AAC it's always 1024 samples in one AAC frame
	spf := 1024
	sr := config.Cfg.StreamSamplerate
	if config.Cfg.StreamAACProfile != "lc" {
		sr = sr / 2
	}
	framesToRead := (sr / spf) + 1

	for {
		sendBegin := time.Now()

		lbuf, err := aac.GetFramesStdin(f, framesToRead)
		if err != nil {
			logger.Log("Error reading data stream", logger.LOG_ERROR)
			cleanUp(err)
			break
		}

		if len(lbuf) <= 0 {
			logger.Log("STDIN from ffmpeg ended", logger.LOG_DEBUG)
			break
		}

		if totalFramesSent == 0 {
			totalTimeBegin = time.Now()
			//stdoutFramesSent = 0
		}

		if err := network.Send(sock, lbuf); err != nil {
			logger.Log("Error sending data stream", logger.LOG_DEBUG)
			cleanUp(err)
			break
		}

		totalFramesSent = totalFramesSent + uint64(framesToRead)
		frames = frames + framesToRead

		timeElapsed := int(float64((time.Now().Sub(totalTimeBegin)).Seconds()) * 1000)
		timeSent := int(float64(totalFramesSent) * float64(spf) / float64(sr) * 1000)
		timeFileElapsed := int(float64((time.Now().Sub(timeFileBegin)).Seconds()) * 1000)

		bufferSent := 0
		if timeSent > timeElapsed {
			bufferSent = timeSent - timeElapsed
		}

		if config.Cfg.UpdateMetadata {
			cuesheet.Update(uint32(timeFileElapsed))
		}

		// calculate the send lag
		sendLag := int(float64((time.Now().Sub(sendBegin)).Seconds()) * 1000)

		if timeElapsed > 1500 {
			logger.Term("Frames: "+strconv.Itoa(frames)+"/"+strconv.Itoa(int(totalFramesSent))+"  Time: "+
				strconv.Itoa(int(timeElapsed))+"/"+strconv.Itoa(int(timeSent))+"ms  Buffer: "+
				strconv.Itoa(int(bufferSent))+"ms  Bps: "+strconv.Itoa(len(lbuf)), logger.LOG_INFO)
		}

		// regulate sending rate
		timePause := 0
		if bufferSent < (config.Cfg.BufferSize - 100) {
			timePause = 900 - sendLag
		} else {
			if bufferSent > config.Cfg.BufferSize {
				timePause = 1100 - sendLag
			} else {
				timePause = 975 - sendLag
			}
		}

		if Abort {
			err := errors.New("Aborted by user")
			cleanUp(err)
			break
		}

		time.Sleep(time.Duration(time.Millisecond) * time.Duration(timePause))
	}
	cmd.Wait()
	logger.Log("ffmpeg is dead. hoy!", logger.LOG_DEBUG)
	//logger.Log(strconv.Itoa(cmd.ProcessState), logger.LOG_DEBUG)
	return res
}
