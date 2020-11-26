package stream

import (
	"bufio"
	"errors"
	"net"
	"os"
	"os/exec"
	"strconv"
	"time"

	"github.com/stunndard/goicy/aac"
	"github.com/stunndard/goicy/config"
	"github.com/stunndard/goicy/cuesheet"
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/metadata"
	"github.com/stunndard/goicy/mpeg"
	"github.com/stunndard/goicy/network"
	"github.com/stunndard/goicy/util"
)

var totalFramesSent uint64
var totalTimeBegin time.Time
var Abort bool

func File(filename string) error {
	var (
		br                  float64
		spf, sr, frames, ch int
		sock                net.Conn
	)

	cleanUp := func(err error) {
		network.Close(sock)
		//totalFramesSent = 0
	}

	logger.Log("Checking file: "+filename+"...", logger.LogInfo)

	var err error
	if config.Cfg.StreamFormat == "mpeg" {
		err = mpeg.GetFileInfo(filename, &br, &spf, &sr, &frames, &ch)
	} else {
		err = aac.GetFileInfo(filename, &br, &spf, &sr, &frames, &ch)
	}
	if err != nil {
		return err
	}

	sock, err = network.ConnectServer(config.Cfg.Host, config.Cfg.Port, br, sr, ch)
	if err != nil {
		logger.Log("Cannot connect to server", logger.LogError)
		return err
	}

	f, err := os.Open(filename)
	if err != nil {
		cleanUp(err)
		return err
	}

	//noinspection GoUnhandledErrorResult
	defer f.Close()

	if config.Cfg.StreamFormat == "mpeg" {
		mpeg.SeekTo1StFrame(*f)
	} else {
		aac.SeekTo1StFrame(*f)
	}

	logger.Log("Streaming file: "+filename+"...", logger.LogInfo)

	cuefile := util.Basename(filename) + ".cue"
	if config.Cfg.UpdateMetadata {
		//noinspection GoUnhandledErrorResult
		go metadata.GetTagsFFMPEG(filename)
		cuesheet.Load(cuefile)
	}

	logger.TermLn("CTRL-C to stop", logger.LogInfo)

	framesSent := 0

	// get number of frames to read in 1 iteration
	framesToRead := (sr / spf) + 1
	timeBegin := time.Now()

	for framesSent < frames {
		sendBegin := time.Now()

		var lbuf []byte
		if config.Cfg.StreamFormat == "mpeg" {
			lbuf, err = mpeg.GetFrames(*f, framesToRead)
		} else {
			lbuf, err = aac.GetFrames(*f, framesToRead)
		}
		if err != nil {
			logger.Log("Error reading data stream", logger.LogError)
			cleanUp(err)
			return err
		}

		if err := network.Send(sock, lbuf); err != nil {
			cleanUp(err)
			logger.Log("Error sending data stream", logger.LogError)
			return err
		}

		framesSent = framesSent + framesToRead

		timeElapsed := int((time.Now().Sub(timeBegin)).Seconds() * 1000)
		timeSent := int(float64(framesSent) * float64(spf) / float64(sr) * 1000)

		bufferSent := 0
		if timeSent > timeElapsed {
			bufferSent = timeSent - timeElapsed
		}

		if config.Cfg.UpdateMetadata {
			cuesheet.Update(uint32(timeElapsed))
		}

		// calculate the send lag
		sendLag := int((time.Now().Sub(sendBegin)).Seconds() * 1000)

		if timeElapsed > 1500 {
			logger.Term("Frames: "+strconv.Itoa(framesSent)+"/"+strconv.Itoa(frames)+"  Time: "+
				strconv.Itoa(timeElapsed/1000)+"/"+strconv.Itoa(timeSent/1000)+"s  Buffer: "+
				strconv.Itoa(bufferSent)+"ms  Frames/Bytes: "+strconv.Itoa(framesToRead)+"/"+strconv.Itoa(len(lbuf)), logger.LogInfo)
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

		time.Sleep(time.Millisecond * time.Duration(timePause))
	}

	// pause to clear up the buffer
	timeBetweenTracks := int(((float64(frames)*float64(spf))/float64(sr))*1000) - int((time.Now().Sub(timeBegin)).Seconds()*1000)
	logger.Log("Pausing for "+strconv.Itoa(timeBetweenTracks)+"ms...", logger.LogDebug)
	time.Sleep(time.Millisecond * time.Duration(timeBetweenTracks))

	return nil
}

func FFMPEG(filename, title string) error {
	var (
		sock         net.Conn
		res          error
		cmd          *exec.Cmd
		sendBegin    time.Time
		stopWatchDog bool
	)

	cleanUp := func(err error) {
		logger.Log("Killing ffmpeg..", logger.LogDebug)
		_ = cmd.Process.Kill()
		network.Close(sock)
		totalFramesSent = 0
		stopWatchDog = true
		res = err
	}

	var err error
	sock, err = network.ConnectServer(config.Cfg.Host, config.Cfg.Port, 0, 0, 0)
	if err != nil {
		logger.Log("Cannot connect to server", logger.LogError)
		return err
	}

	cmdArgs := []string{""}
	profile := ""
	if config.Cfg.StreamFormat == "mpeg" {
		profile = "MPEG"
		if config.Cfg.StreamReencode {
			cmdArgs = []string{
				"-i", filename,
				"-c:a", "libmp3lame",
				"-b:a", strconv.Itoa(config.Cfg.StreamBitrate),
				"-cutoff", "20000",
				"-ar", strconv.Itoa(config.Cfg.StreamSamplerate),
				"-ac", strconv.Itoa(config.Cfg.StreamChannels),
				"-f", "mp3",
				"-write_xing", "0",
				"-id3v2_version", "0",
				"-loglevel", "fatal",
				"-",
			}
		} else {
			cmdArgs = []string{
				"-i", filename,
				"-c:a", "copy",
				"-f", "mp3",
				"-write_xing", "0",
				"-id3v2_version", "0",
				"-loglevel", "fatal",
				"-",
			}
		}
	} else {
		if config.Cfg.StreamAACProfile == "lc" {
			profile = "aac_low"
		} else if config.Cfg.StreamAACProfile == "he" {
			profile = "aac_he"
		} else {
			profile = "aac_he_v2"
		}
		if config.Cfg.StreamReencode {
			cmdArgs = []string{
				"-i", filename,
				"-c:a", "libfdk_aac",
				"-profile:a", profile,
				"-b:a", strconv.Itoa(config.Cfg.StreamBitrate),
				"-cutoff", "20000",
				"-ar", strconv.Itoa(config.Cfg.StreamSamplerate),
				"-ac", strconv.Itoa(config.Cfg.StreamChannels),
				"-f", "adts",
				"-loglevel", "fatal",
				"-",
			}
		} else {
			cmdArgs = []string{
				"-i", filename,
				"-c:a", "copy",
				"-f", "adts",
				"-loglevel", "fatal",
				"-",
			}
		}
	}

	logger.Log("Starting ffmpeg: "+config.Cfg.FFMPEGPath, logger.LogDebug)
	if config.Cfg.StreamReencode {
		logger.Log("Format         : "+profile, logger.LogDebug)
		logger.Log("Bitrate        : "+strconv.Itoa(config.Cfg.StreamBitrate), logger.LogDebug)
		logger.Log("Samplerate     : "+strconv.Itoa(config.Cfg.StreamSamplerate), logger.LogDebug)
	} else {
		logger.Log("Format        : source, no reencoding", logger.LogDebug)
	}

	cmd = exec.Command(config.Cfg.FFMPEGPath, cmdArgs...)

	f, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		logger.Log("Error starting ffmpeg", logger.LogError)
		logger.Log(err.Error(), logger.LogError)
		return err
	}

	// log stderr output from ffmpeg
	go func() {
		in := bufio.NewScanner(stderr)
		for in.Scan() {
			logger.Log("FFMPEG: "+in.Text(), logger.LogDebug)
		}
	}()
	// watchdog to kill stalled ffmpeg
	go func() {
		//logger.Log("watchdog started", logger.LOG_DEBUG)
		for {
			time.Sleep(time.Duration(time.Millisecond) * time.Duration(1000))
			if stopWatchDog {
				//logger.Log("watchdog stopped", logger.LOG_DEBUG)
				break
			}
			timeDataSeen := int(float64((time.Now().Sub(sendBegin)).Seconds()) * 1000)
			if timeDataSeen > 8000 {
				logger.Log("ffmpeg stalled, killing... "+strconv.Itoa(int(timeDataSeen))+"ms", logger.LogError)
				cmd.Process.Kill()
				break
			}
		}
	}()

	logger.Log("Streaming file: "+filename+"...", logger.LOG_INFO)

	cuefile := util.Basename(filename) + ".cue"
	if config.Cfg.UpdateMetadata {
		if title == "" {
			//noinspection GoUnhandledErrorResult
			go metadata.GetTagsFFMPEG(filename)
			cuesheet.Load(cuefile)
		} else {
			//noinspection GoUnhandledErrorResult
			go metadata.SendMetadata(title)
		}
	}

	logger.TermLn("CTRL-C to stop", logger.LogInfo)

	frames := 0
	timeFileBegin := time.Now()

	sr := 0
	spf := 0
	framesToRead := 1

	for {
		sendBegin := time.Now()

		var lbuf []byte
		if config.Cfg.StreamFormat == "mpeg" {
			lbuf, err = mpeg.GetFramesStdin(f, framesToRead)
			if framesToRead == 1 {
				if len(lbuf) < 4 {
					logger.Log("Error reading data stream", logger.LogError)
					cleanUp(err)
					break
				}
				sr = mpeg.GetSR(lbuf[0:4])
				if sr == 0 {
					logger.Log("Erroneous MPEG sample rate from data stream", logger.LogError)
					cleanUp(err)
					break
				}
				spf = mpeg.GetSPF(lbuf[0:4])
				framesToRead = (sr / spf) + 1
				mbuf, _ := mpeg.GetFramesStdin(f, framesToRead-1)
				lbuf = append(lbuf, mbuf...)
			}
		} else {
			lbuf, err = aac.GetFramesStdin(f, framesToRead)
			if framesToRead == 1 {
				if len(lbuf) < 7 {
					logger.Log("Error reading data stream", logger.LogError)
					cleanUp(err)
					break
				}
				sr = aac.GetSR(lbuf[0:7])
				if sr == 0 {
					logger.Log("Erroneous AAC sample rate from data stream", logger.LogError)
					cleanUp(err)
					break
				}
				spf = aac.GetSPF(lbuf[0:7])
				framesToRead = (sr / spf) + 1
				mbuf, _ := aac.GetFramesStdin(f, framesToRead-1)
				lbuf = append(lbuf, mbuf...)
			}
		}

		if err != nil {
			logger.Log("Error reading data stream", logger.LogError)
			cleanUp(err)
			break
		}

		if len(lbuf) <= 0 {
			logger.Log("STDIN from ffmpeg ended", logger.LOG_DEBUG)
			stopWatchDog = true
			break
		}

		if totalFramesSent == 0 {
			totalTimeBegin = time.Now()
			//stdoutFramesSent = 0
		}

		if err := network.Send(sock, lbuf); err != nil {
			logger.Log("Error sending data stream", logger.LogError)
			cleanUp(err)
			break
		}

		totalFramesSent = totalFramesSent + uint64(framesToRead)
		frames = frames + framesToRead

		timeElapsed := int((time.Now().Sub(totalTimeBegin)).Seconds() * 1000)
		timeSent := int(float64(totalFramesSent) * float64(spf) / float64(sr) * 1000)
		timeFileElapsed := int((time.Now().Sub(timeFileBegin)).Seconds() * 1000)

		bufferSent := 0
		if timeSent > timeElapsed {
			bufferSent = timeSent - timeElapsed
		}

		if config.Cfg.UpdateMetadata {
			cuesheet.Update(uint32(timeFileElapsed))
		}

		// calculate the send lag
		sendLag := int((time.Now().Sub(sendBegin)).Seconds() * 1000)

		if timeElapsed > 1500 {
			logger.Term("Frames: "+strconv.Itoa(frames)+"/"+strconv.Itoa(int(totalFramesSent))+"  Time: "+
				strconv.Itoa(timeElapsed/1000)+"/"+strconv.Itoa(timeSent/1000)+"s  Buffer: "+
				strconv.Itoa(bufferSent)+"ms  Frames/Bytes: "+strconv.Itoa(framesToRead)+"/"+strconv.Itoa(len(lbuf)),
				logger.LogInfo)
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
			break
		}

		time.Sleep(time.Millisecond * time.Duration(timePause))
	}
	_ = cmd.Wait()
	logger.Log("ffmpeg is dead. hoy!", logger.LogDebug)

	//logger.Log(strconv.Itoa(cmd.ProcessState), logger.LOG_DEBUG)
	return res
}
