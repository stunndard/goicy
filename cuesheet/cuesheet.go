package cuesheet

import (
	"bufio"
	"github.com/bgroupe/goicy/logger"
	"github.com/bgroupe/goicy/metadata"
	"io"
	"os"
	"strconv"
	"strings"
)

type cueEntry struct {
	Title  string
	Artist string
	Time   uint32 // in milliseconds
}

var cueEntries []cueEntry
var idx int
var loaded bool

func get_value(entry, key string) string {
	s := string(entry[len(key)+1:])
	if string(s[0]) == "\"" {
		s = s[1 : len(s)-1]
	}
	//fmt.Println(key, "=", s)
	return s
}

func get_time(time string) uint32 {
	ttime := time[0 : len(time)-3] //Copy(time, 1, length(time) - 3);

	s := ttime[0 : len(ttime)-3]
	z, _ := strconv.Atoi(s)
	s = ttime[len(ttime)-2:]
	x, _ := strconv.Atoi(s)
	//fmt.Println(z)
	//fmt.Println(x)
	return uint32(z*60000 + x*1000)
}

func isUpdate(time uint32) bool {
	res := false
	//fmt.Println("isupdate: ", time)
	if idx < len(cueEntries) {
		if time >= cueEntries[idx].Time {
			idx = idx + 1
			res = true
		}
	}
	return res
}

func get_tags() (artist, title string) {
	if idx > 0 {
		artist = cueEntries[idx-1].Artist
		title = cueEntries[idx-1].Title
	}
	return artist, title
}

func Update(time uint32) {
	if !loaded {
		return
	}
	if isUpdate(time) {
		md := metadata.FormatMetadata(get_tags())
		go metadata.SendMetadata(md)
	}
}

func Load(cuefile string) bool {
	var entry string

	loaded = false
	idx = 0
	cueEntries = nil

	f, err := os.Open(cuefile)
	if err != nil {
		//fmt.Println("error opening file ", err)
		return false
	}
	defer f.Close()
	r := bufio.NewReader(f)
	for err != io.EOF {
		entry, err = r.ReadString(0x0A) // 0x0A separator = newline
		entry = strings.Trim(entry, "\r\n ")
		for (err != io.EOF) && (entry[0:5] == "TRACK") {
			cueEntries = append(cueEntries, cueEntry{})
			entry, err = r.ReadString(0x0A)
			entry = strings.Trim(entry, "\r\n ")
			for (err != io.EOF) && (entry[0:5] != "TRACK") {
				if entry[0:5] == "TITLE" {
					cueEntries[idx].Title = get_value(entry, "TITLE")
				}
				if entry[0:9] == "PERFORMER" {
					cueEntries[idx].Artist = get_value(entry, "PERFORMER")
				}
				if entry[0:8] == "INDEX 01" {
					cueEntries[idx].Time = get_time(get_value(entry, "INDEX 01"))
				}
				if (err != nil) && (err == io.EOF) {
					break
				}
				entry, err = r.ReadString(0x0A)
				entry = strings.Trim(entry, "\r\n ")
			}
			idx = idx + 1
		}
	}
	loaded = idx > 0
	idx = 0
	if loaded {
		logger.Log("Loaded cuesheet: "+cuefile, logger.LOG_INFO)
	}
	return loaded
}
