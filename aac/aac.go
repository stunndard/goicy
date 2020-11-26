package aac

import (
	"io"
	"os"
	"strconv"

	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/util"
)

var sftable = [...]int{
	96000, 88200, 64000, 48000,
	44100, 32000, 24000, 22050,
	16000, 12000, 11025, 8000,
	7350, 0, 0, 0}

func isValidFrameHeader(header []byte) (int, bool) {
	// check for valid syncowrd
	syncword := (uint16(header[0]) << 4) | (uint16(header[1]) >> 4)
	if syncword != 0x0FFF {
		return 0, false
	}

	// get and check the profile
	profile := (header[2] & 0x0C0) >> 6
	if profile == 3 {
		return 0, false
	}

	// get and check the 'sampling_frequency_index':
	sfindex := (header[2] & 0x03C) >> 2
	if sftable[sfindex] == 0 {
		return 0, false
	}

	// get and check "channel configuration"
	ch := int(((header[2] & 0x01) << 2) | ((header[3] & 0x0C0) >> 6))
	if (ch < 1) || (ch > 7) {
		return 0, false
	}

	frameLength := ((int(header[3]) & 0x03) << 11) | (int(header[4]) << 3) | ((int(header[5]) & 0x0E0) >> 5)
	if (frameLength < 7) || (frameLength > 5000) {
		return 0, false
	}

	//(valid frame, len= ', frameLength);
	return frameLength, true
}

func GetSPF(_ []byte) int {
	return 1024
}

func GetSR(header []byte) int {
	// get and check the 'sampling_frequency_index':
	sfindex := (header[2] & 0x03C) >> 2
	return sftable[sfindex]
}

func SeekTo1StFrame(f os.File) int64 {

	buf := make([]byte, 50000)
	f.ReadAt(buf, 0)

	// skip ID3V2 at the beginning of file
	var ID3Length int64 = 0
	for id3 := string(buf[0:3]); id3 == "ID3"; {
		//major := byte(buf[4])
		//minor := byte(buf[5])
		//flags := buf[6]
		ID3Length = ID3Length + (int64(buf[6])<<21 | int64(buf[7])<<14 | int64(buf[8])<<7 | int64(buf[9])) + 10
		f.ReadAt(buf, ID3Length)
		id3 = string(buf[0:3])
	}

	pos := int64(-1)

	for i := 0; i < len(buf); i++ {
		if (buf[i] == 0xFF) && ((buf[i+1] & 0xF0) == 0xF0) {
			if len(buf)-i < 10 {
				break
			}
			aacHeader := buf[i : i+7]

			if n, ok := isValidFrameHeader(aacHeader); ok {
				if i+n+7 >= len(buf) {
					break
				}
				aacHeader = buf[i+n : i+n+7]
				if m, ok := isValidFrameHeader(aacHeader); ok {
					if i+n+m+7 >= len(buf) {
						break
					}
					aacHeader = buf[i+n+m : i+n+m+7]
					if _, ok := isValidFrameHeader(aacHeader); !ok {
						continue
					}
				}
				pos = int64(i) + ID3Length
				f.Seek(pos, 0)
				break
			}
		}
	}
	return pos
}

func GetFrames(f os.File, framesToRead int) ([]byte, error) {

	var framesRead, bytesRead = 0, 0
	var headers = make([]byte, 7)
	var inSync = true
	var numBytesToRead = 0
	var buf []byte
	var err error

	for framesRead < framesToRead {

		bytesRead, err = f.Read(headers)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		}
		if bytesRead < len(headers) {
			//input file has ended
			break
		}

		if _, ok := isValidFrameHeader(headers); !ok {
			if inSync {
				pos, _ := f.Seek(0, 1)
				logger.Log("Bad AAC frame at offset "+strconv.Itoa(int(pos-7))+
					", resyncing...", logger.LogDebug)
			}
			f.Seek(-6, 1)
			inSync = false
			continue
		}

		// from now on, the frame is considered valid
		if !inSync {
			pos, _ := f.Seek(0, 1)
			logger.Log("Resynced at offset "+strconv.Itoa(int(pos-7)), logger.LogDebug)
		}
		inSync = true

		// copy frame header to out buffer
		buf = append(buf, headers...)

		// extract important fields from aac headers:
		FrameLength := ((int(headers[3]) & 0x03) << 11) | (int(headers[4]) << 3) | ((int(headers[5]) & 0x0E0) >> 5)

		if FrameLength > len(headers) {
			numBytesToRead = FrameLength - len(headers)
		} else {
			numBytesToRead = 0
		}

		// read raw frame data
		lbuf := make([]byte, numBytesToRead)
		bytesRead, err = f.Read(lbuf)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		}

		buf = append(buf, lbuf[0:bytesRead]...)

		if bytesRead < numBytesToRead {
			// the input file has ended
			break
		}
		framesRead = framesRead + 1
	}

	return buf, nil
}

func GetFramesStdin(f io.ReadCloser, framesToRead int) ([]byte, error) {

	var framesRead, bytesRead = 0, 0
	var headers = make([]byte, 7)
	//var inSync bool = true
	var numBytesToRead = 0
	var buf []byte
	var err error

	for framesRead < framesToRead {

		bytesRead, err = f.Read(headers)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		}
		if bytesRead < len(headers) {
			//input file has ended
			break
		}

		if _, ok := isValidFrameHeader(headers); !ok {
			logger.Log("Bad AAC frame encountered", logger.LogDebug)
		}

		// copy frame header to out buffer
		buf = append(buf, headers...)

		// extract important fields from aac headers:
		FrameLength := ((int(headers[3]) & 0x03) << 11) | (int(headers[4]) << 3) | ((int(headers[5]) & 0x0E0) >> 5)

		if FrameLength > len(headers) {
			numBytesToRead = FrameLength - len(headers)
		} else {
			numBytesToRead = 0
		}

		// read raw frame data
		lbuf := make([]byte, numBytesToRead)
		bytesRead, err = f.Read(lbuf)
		if err != nil {
			if err != io.EOF {
				return nil, err
			}
		}

		buf = append(buf, lbuf[0:bytesRead]...)

		if bytesRead < numBytesToRead {
			// the input file has ended
			break
		}
		framesRead = framesRead + 1
	}
	return buf, nil
}

// gets information about AAC file
func GetFileInfo(filename string, br *float64, spf, sr, frames, ch *int) error {

	if ok := util.FileExists(filename); !ok {
		err := new(util.FileError)
		err.Msg = "File doesn't exist"
		return err
	}

	// open file
	f, err := os.Open(filename)
	if err != nil {
		err := new(util.FileError)
		err.Msg = "Cannot open file"
		return err
	}

	defer f.Close()

	firstFramePos := SeekTo1StFrame(*f)
	if firstFramePos == -1 {
		err := new(util.FileError)
		err.Msg = "Couldn't find AAC frame"
		return err
	}

	logger.Log("First frame found at offset: "+strconv.Itoa(int(firstFramePos)), logger.LogDebug)

	// now having opened the input file, read the fixed header of the
	// first frame, to get the audio stream's parameters:
	fixheader := make([]byte, 4)

	var sfindex byte = 0
	frame := 1

	if n, err := f.Read(fixheader); (n == len(fixheader)) && (err == nil) {
		// check the 'syncword'
		if (fixheader[0] != 0x0FF) && ((fixheader[1] & 0x0F0) != 0x0F0) {
			err := new(util.FileError)
			err.Msg = "Bad \"syncword\" at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check the profile
		profile := (fixheader[2] & 0x0C0) >> 6
		if profile == 3 {
			err := new(util.FileError)
			err.Msg = "Bad (reserved) \"profile\":3 at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check the 'sampling_frequency_index':
		sfindex = (fixheader[2] & 0x3C) >> 2
		if sftable[sfindex] == 0 {
			err := new(util.FileError)
			err.Msg = "Bad \"sampling_frequency_index\" at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check "channel configuration"
		*ch = int(((fixheader[2] & 0x01) << 2) | ((fixheader[3] & 0x0C0) >> 6))
		if (*ch < 1) || (*ch > 7) {
			err := new(util.FileError)
			err.Msg = "Bad \"channel configuration\" at frame # " + strconv.Itoa(frame)
			return err
		}

		f.Seek(firstFramePos, 0)

		headers := make([]byte, 7)
		var numBytesToRead = 0

		for {
			if n, err = f.Read(headers); (n < len(headers)) || (err != nil) {
				break
			}
			protectionAbsent := headers[1] & 0x01
			var frameLength = ((int(headers[3]) & 0x03) << 11) | (int(headers[4]) << 3) | ((int(headers[5]) & 0xE0) >> 5)

			if _, ok := isValidFrameHeader(headers); !ok {
				f.Seek(-6, 1)
				continue
			} else {
				frame++
			}
			if frameLength > len(headers) {
				numBytesToRead = frameLength - len(headers)
			} else {
				numBytesToRead = 0
			}
			if protectionAbsent == 0 {
				f.Seek(2, 1)
				if numBytesToRead > 2 {
					numBytesToRead -= 2
				} else {
					numBytesToRead = 0
				}
			}

			// skip raw frame data
			f.Seek(int64(numBytesToRead), 1)
		}
	}
	finfo, _ := f.Stat()
	fsize := finfo.Size()

	*spf = 1024
	*sr = sftable[sfindex]
	*frames = frame - 1
	nsamples := 1024 * *frames
	playtime := nsamples / *sr
	*br = float64(fsize-firstFramePos) / float64(playtime)
	*br = *br * 8 / 1000

	logger.Log("frames    : "+strconv.Itoa(*frames), logger.LogDebug)
	logger.Log("samplerate: "+strconv.Itoa(*sr)+" Hz", logger.LogDebug)
	logger.Log("channels  : "+strconv.Itoa(*ch), logger.LogDebug)
	logger.Log("playtime  : "+strconv.Itoa(playtime)+" sec", logger.LogDebug)
	logger.Log("bitrate   : "+strconv.Itoa(int(*br))+" kbps (average)", logger.LogDebug)

	return nil
}
