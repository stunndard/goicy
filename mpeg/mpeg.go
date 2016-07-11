package mpeg

import (
	"github.com/stunndard/goicy/logger"
	"github.com/stunndard/goicy/util"
	"io"
	"os"
	"strconv"
)

func isValidFrameHeader(header []byte) (int, bool) {

	if (header[0] != 0x0FF) && ((header[1] & 0x0E0) != 0x0E0) {
		return 0, false
	}

	// get and check the mpeg version
	mpegver := (uint16(header[1]) & 0x18) >> 3
	if mpegver == 1 || mpegver > 3 {
		return 0, false
	}

	// get and check mpeg layer
	layer := (header[1] & 0x06) >> 1
	if layer == 0 || layer > 3 {
		return 0, false
	}

	// get and check bitreate index
	brindex := (header[2] & 0x0F0) >> 4
	if brindex > 15 {
		return 0, false
	}

	// get and check the 'sampling_rate_index':
	srindex := (header[2] & 0x0C) >> 2
	if srindex >= 3 {
		return 0, false
	}

	return 0, true
}

func GetSPF(header []byte) int {
	// get and check the mpeg version
	mpegver := byte((header[1] & 0x18) >> 3)

	// get and check mpeg layer
	layer := byte((header[1] & 0x06) >> 1)

	spf := 0
	switch mpegver {
	case 3: // mpeg 1
		if layer == 3 { // layer1
			spf = 384
		} else {
			spf = 1152 // layer2 & layer3
		}
	case 2, 0: // mpeg2 & mpeg2.5
		switch layer {
		case 3: // layer1
			spf = 384
		case 2: // layer2
			spf = 1152
		default:
			spf = 576 // layer 3
		}
	}
	return spf
}

func getFrameSize(header []byte) int {
	var sr, bitrate uint32
	var res int

	brtable := [...]uint32{
		0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0,
		0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0,
		0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0,
		0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0,
		0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}
	srtable := [...]uint32{
		44100, 48000, 32000, 0, // mpeg1
		22050, 24000, 16000, 0, // mpeg2
		11025, 12000, 8000, 0} // mpeg2.5

	// get and check the mpeg version
	mpegver := byte((header[1] & 0x18) >> 3)
	if mpegver == 1 || mpegver > 3 {
		return 0
	}

	// get and check mpeg layer
	layer := byte((header[1] & 0x06) >> 1)
	if layer == 0 || layer > 3 {
		return 0
	}

	brindex := byte((header[2] & 0x0F0) >> 4)

	if mpegver == 3 && layer == 3 {
		// mpeg1, layer1
		bitrate = brtable[brindex]
	}
	if mpegver == 3 && layer == 2 {
		// mpeg1, layer2
		bitrate = brtable[brindex+16]
	}
	if mpegver == 3 && layer == 1 {
		// mpeg1, layer3
		bitrate = brtable[brindex+32]
	}
	if (mpegver == 2 || mpegver == 0) && layer == 3 {
		// mpeg2, 2.5, layer1
		bitrate = brtable[brindex+48]
	}
	if (mpegver == 2 || mpegver == 0) && (layer == 2 || layer == 1) {
		//mpeg2, layer2 or layer3
		bitrate = brtable[brindex+64]
	}
	bitrate = bitrate * 1000
	padding := int(header[2]&0x02) >> 1

	// get and check the 'sampling_rate_index':
	srindex := byte((header[2] & 0x0C) >> 2)
	if srindex >= 3 {
		return 0
	}
	if mpegver == 3 {
		sr = srtable[srindex]
	}
	if mpegver == 2 {
		sr = srtable[srindex+4]
	}
	if mpegver == 0 {
		sr = srtable[srindex+8]
	}

	switch mpegver {
	case 3: // mpeg1
		if layer == 3 { // layer1
			res = (int(12*bitrate/sr) * 4) + (padding * 4)
		}
		if layer == 2 || layer == 1 {
			// layer 2 & 3
			res = int(144*bitrate/sr) + padding
		}

	case 2, 0: //mpeg2, mpeg2.5
		if layer == 3 { // layer1
			res = (int(12*bitrate/sr) * 4) + (padding * 4)
		}
		if layer == 2 { // layer2
			res = int(144*bitrate/sr) + padding
		}
		if layer == 1 { // layer3
			res = int(72*bitrate/sr) + padding
		}
	}
	return res
}

func SeekTo1StFrame(f os.File) int {

	buf := make([]byte, 100000)
	f.ReadAt(buf, 0)

	j := int64(-1)

	for i := 0; i < len(buf); i++ {
		if (buf[i] == 0xFF) && ((buf[i+1] & 0xE0) == 0xE0) {
			if len(buf)-i < 10 {
				break
			}
			mpx_header := buf[i : i+4]
			if _, ok := isValidFrameHeader(mpx_header); ok {
				if framelength := getFrameSize(mpx_header); framelength > 0 {
					if i+framelength+4 > len(buf) {
						break
					}
					mpx_header = buf[i+framelength : i+framelength+4]
					if _, ok := isValidFrameHeader(mpx_header); ok {
						j = int64(i)
						f.Seek(j, 0)
						break
					}
				}
			}
		}
	}
	return int(j)
}

func GetFrames(f os.File, framesToRead int) ([]byte, error) {
	var framesRead, bytesRead int = 0, 0
	var headers []byte = make([]byte, 4)
	var headers2 []byte = make([]byte, 4)
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
				logger.Log("Bad MPEG frame at offset "+strconv.Itoa(int(pos-4))+
					", resyncing...", logger.LOG_DEBUG)
			}
			f.Seek(-3, 1)
			inSync = false
			continue
		}

		framelength := getFrameSize(headers)
		if framelength == 0 || framelength > 5000 {
			if inSync {
				pos, _ := f.Seek(0, 1)
				logger.Log("Bad MPEG frame at offset "+strconv.Itoa(int(pos-4))+
					", resyncing...", logger.LOG_DEBUG)
			}
			f.Seek(-3, 1)
			inSync = false
			continue
		}

		if framelength > len(headers) {
			numBytesToRead = framelength - len(headers) // + crc
		} else {
			numBytesToRead = 0
		}

		oldpos, _ := f.Seek(0, 1)
		br, _ := f.Seek(int64(numBytesToRead), 1)
		bytesRead = int(br - oldpos)
		bbr, _ := f.Read(headers2)
		bytesRead = bytesRead + int(bbr)
		f.Seek(int64(-bytesRead), 1)
		if _, ok := isValidFrameHeader(headers2); !ok {
			if inSync {
				pos, _ := f.Seek(0, 1)
				logger.Log("Bad MPEG frame at offset "+strconv.Itoa(int(pos-4))+
					", resyncing...", logger.LOG_DEBUG)
			}
			f.Seek(-3, 1)
			inSync = false
			continue
		}

		// from now on, frame is considered valid
		if !inSync {
			pos, _ := f.Seek(0, 1)
			logger.Log("Resynced at offset "+strconv.Itoa(int(pos-4)), logger.LOG_DEBUG)
		}
		inSync = true

		// copy frame header to out buffer
		buf = append(buf, headers...)

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
	var headers []byte = make([]byte, 4)
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
			logger.Log("Bad MPEG frame encountered", logger.LOG_DEBUG)
		}

		// copy frame header to out buffer
		buf = append(buf, headers...)

		// get frame size from MPEG header
		frameLength := getFrameSize(headers)

		if frameLength > len(headers) {
			numBytesToRead = frameLength - len(headers)
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

// gets information about MPEG file
func GetFileInfo(filename string, br *float64, spf, sr, frames, ch *int) error {
	var mpegver, layer byte

	srtable := [...]uint32{
		44100, 48000, 32000, 0, // mpeg1
		22050, 24000, 16000, 0, // mpeg2
		11025, 12000, 8000, 0} // mpeg2.5
	brtable := [...]uint32{
		0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 320, 352, 384, 416, 448, 0,
		0, 32, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 384, 0,
		0, 32, 40, 48, 56, 64, 80, 96, 112, 128, 160, 192, 224, 256, 320, 0,
		0, 32, 48, 56, 64, 80, 96, 112, 128, 144, 160, 176, 192, 224, 256, 0,
		0, 8, 16, 24, 32, 40, 48, 56, 64, 80, 96, 112, 128, 144, 160, 0}

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
		err.Msg = "Couldn't find MPEG frame"
		return err
	}

	logger.Log("First frame found at offset: "+strconv.Itoa(j), logger.LOG_DEBUG)

	// now having opened the input file, read the fixed header of the
	// first frame, to get the audio stream's parameters:
	header := make([]byte, 4)

	var srindex byte = 0
	frame := 1

	if n, err := f.Read(header); (n == len(header)) && (err == nil) {
		// check the 'syncword'
		if (header[0] != 0x0FF) && ((header[1] & 0x0E0) != 0x0E0) {
			err := new(util.FileError)
			err.Msg = "Bad \"frame sync\" at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check the mpeg version
		mpegver = byte((header[1] & 0x018) >> 3)
		if mpegver == 1 {
			err := new(util.FileError)
			err.Msg = "Bad (reserved) mpeg version at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check mpeg layer
		layer = byte((header[1] & 0x06) >> 1)
		if layer == 0 {
			err := new(util.FileError)
			err.Msg = "Bad (reserved) mpeg layer at frame # " + strconv.Itoa(frame)
			return err
		}

		// get and check the 'sampling_rate_index':
		srindex = (header[2] & 0x0C) >> 2
		if srtable[srindex] == 0 {
			err := new(util.FileError)
			err.Msg = "Bad sampling_frequency_index at frame # " + strconv.Itoa(frame)
			return err
		}
		if mpegver == 3 {
			// mpeg1
			*sr = int(srtable[srindex])
		}
		if mpegver == 2 {
			// mpeg2
			*sr = int(srtable[srindex+4])
		}
		if mpegver == 0 {
			// mpeg2.5
			*sr = int(srtable[srindex+8])
		}

		// get and check "channel configuration"
		*ch = int(header[3]&0x0C0) >> 6

		f.Seek(int64(j), 0)

		var numBytesToRead int = 0

		for {
			if n, err = f.Read(header); (n < len(header)) || (err != nil) {
				// the input file has ended
				break
			}

			// get and check bitrate
			brindex := (header[2] & 0x0F0) >> 4
			if brindex == 0 || brindex == 0x0F {
				f.Seek(-3, 1)
				continue
			}

			if mpegver == 3 && layer == 3 {
				// mpeg1, layer1
				*br = float64(brtable[brindex])
			}
			if mpegver == 3 && layer == 2 {
				// mpeg1, layer2
				*br = float64(brtable[brindex+16])
			}
			if mpegver == 3 && layer == 1 {
				// mpeg1, layer3
				*br = float64(brtable[brindex+32])
			}
			if (mpegver == 2 || mpegver == 0) && layer == 3 {
				// mpeg2, 2.5, layer1
				*br = float64(brtable[brindex+48])
			}
			if (mpegver == 2 || mpegver == 0) && (layer == 2 || layer == 1) {
				//mpeg2, layer2 or layer3
				*br = float64(brtable[brindex+64])
			}
			*br = *br * 1000
			//padding := (header[2] & 0x02) >> 1

			framelength := getFrameSize(header)
			if _, ok := isValidFrameHeader(header); !ok || framelength == 0 || framelength > 5000 {
				f.Seek(-3, 1)
				continue
			} else {
				frame = frame + 1
			}

			if framelength > len(header) {
				numBytesToRead = framelength - len(header)
			} else {
				numBytesToRead = 0
			}

			//skip raw frame data
			f.Seek(int64(numBytesToRead), 1)
		}
	}
	finfo, _ := f.Stat()
	fsize := finfo.Size()

	*spf = GetSPF(header)

	*frames = frame - 1
	nsamples := (*spf) * (*frames)
	playtime := float64(nsamples / *sr)
	*br = float64(fsize) / playtime
	*br = *br * 8 / 1000

	var smpegver string
	switch mpegver {
	case 3:
		smpegver = "MPEG 1"
	case 2:
		smpegver = "MPEG 2"
	case 0:
		smpegver = "MPEG 2.5"
	}
	var slayer string
	switch layer {
	case 3:
		slayer = "Layer I"
	case 2:
		slayer = "Layer II"
	case 1:
		slayer = "Layer III"
	}
	var sch string
	switch *ch {
	case 0:
		sch = "Stereo"
	case 1:
		sch = "Joint Stereo"
	case 2:
		sch = "Dual Channel"
	case 3:
		sch = "Mono"
	}
	if *ch == 0 || *ch == 1 || *ch == 2 {
		*ch = 2
	} else {
		*ch = 1
	}

	logger.Log("spf       : "+strconv.Itoa(*spf), logger.LOG_DEBUG)
	logger.Log("format    : "+smpegver+" "+slayer, logger.LOG_DEBUG)
	logger.Log("frames    : "+strconv.Itoa(*frames), logger.LOG_DEBUG)
	logger.Log("samplerate: "+strconv.Itoa(*sr)+" Hz", logger.LOG_DEBUG)
	logger.Log("channels  : "+strconv.Itoa(*ch)+" ("+sch+")", logger.LOG_DEBUG)
	logger.Log("playtime  : "+strconv.Itoa(int(playtime))+" sec", logger.LOG_DEBUG)
	logger.Log("bitrate   : "+strconv.Itoa(int(*br))+" kbps (average)", logger.LOG_DEBUG)

	return nil

}
