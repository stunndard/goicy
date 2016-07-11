package aac

import (
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/util"
	"io"
	"os"
	"strconv"
)

func isValidFrameHeader(header []byte) (int, bool) {
	sftable := [...]int{
		96000, 88200, 64000, 48000,
		44100, 32000, 24000, 22050,
		16000, 12000, 11025, 8000,
		7350, 0, 0, 0}

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

	frame_length := ((int(header[3]) & 0x03) << 11) | (int(header[4]) << 3) | ((int(header[5]) & 0x0E0) >> 5)
	if (frame_length < 7) || (frame_length > 5000) {
		return 0, false
	}

	//(valid frame, len= ', frame_length);
	return frame_length, true
}

func GetSPF(header []byte) int {
	return 1024
}

func SeekTo1StFrame(f os.File) int {

	buf := make([]byte, 5000)
	f.ReadAt(buf, 0)

	j := int64(-1)
	for i := 0; i < len(buf); i++ {
		if (buf[i] == 0xFF) && ((buf[i+1] & 0xF0) == 0xF0) {
			if len(buf)-i < 10 {
				break
			}
			aac_header := buf[i : i+7]

			if n, ok := isValidFrameHeader(aac_header); ok {
				if i+n+7 >= len(buf) {
					break
				}
				aac_header = buf[i+n : i+n+7]
				if m, ok := isValidFrameHeader(aac_header); ok {
					if i+n+m+7 >= len(buf) {
						break
					}
					aac_header = buf[i+n+m : i+n+m+7]
					if _, ok := isValidFrameHeader(aac_header); !ok {
						continue
					}
				}

				j = int64(i)
				f.Seek(j, 0)
				break
			}
		}
	}
	return int(j)
}

func GetFrames(f os.File, framesToRead int) ([]byte, error) {

	var framesRead, bytesRead int = 0, 0
	var headers []byte = make([]byte, 7)
	var inSync bool = true
	var numBytesToRead int = 0
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
					", resyncing...", logger.LOG_DEBUG)
			}
			f.Seek(-6, 1)
			inSync = false
			continue
		}

		// from now on, the frame is considered valid
		if !inSync {
			pos, _ := f.Seek(0, 1)
			logger.Log("Resynced at offset "+strconv.Itoa(int(pos-7)), logger.LOG_DEBUG)
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

	var framesRead, bytesRead int = 0, 0
	var headers []byte = make([]byte, 7)
	//var inSync bool = true
	var numBytesToRead int = 0
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
			logger.Log("Bad AAC frame encountered", logger.LOG_DEBUG)
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

	sftable := [...]int{96000, 88200, 64000, 48000,
		44100, 32000, 24000, 22050,
		16000, 12000, 11025, 8000,
		7350, 0, 0, 0}

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

	j := SeekTo1StFrame(*f)
	if j == -1 {
		err := new(util.FileError)
		err.Msg = "Couldn't find AAC frame"
		return err
	}

	logger.Log("First frame found at offset: "+strconv.Itoa(j), logger.LOG_DEBUG)

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

		f.Seek(int64(j), 0)

		headers := make([]byte, 7)
		var numBytesToRead int = 0

		for {
			if n, err = f.Read(headers); (n < len(headers)) || (err != nil) {
				break
			}
			protection_absent := headers[1] & 0x01
			var frame_length int = (((int(headers[3]) & 0x03) << 11) | (int(headers[4]) << 3) | ((int(headers[5]) & 0xE0) >> 5))

			if _, ok := isValidFrameHeader(headers); !ok {
				f.Seek(-6, 1)
				continue
			} else {
				frame++
			}
			if frame_length > len(headers) {
				numBytesToRead = frame_length - len(headers)
			} else {
				numBytesToRead = 0
			}
			if protection_absent == 0 {
				f.Seek(2, 1)
				if numBytesToRead > 2 {
					numBytesToRead -= 2
				} else {
					numBytesToRead = 0
				}
			}

			//read or skip raw frame data
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
	*br = float64(fsize / int64(playtime))
	*br = *br * 8 / 1000

	logger.Log("frames    : "+strconv.Itoa(*frames), logger.LOG_DEBUG)
	logger.Log("samplerate: "+strconv.Itoa(*sr)+" Hz", logger.LOG_DEBUG)
	logger.Log("channels  : "+strconv.Itoa(*ch), logger.LOG_DEBUG)
	logger.Log("playtime  : "+strconv.Itoa(playtime)+" sec", logger.LOG_DEBUG)
	logger.Log("bitrate   : "+strconv.Itoa(int(*br))+" kbps (average)", logger.LOG_DEBUG)

	return nil
}
