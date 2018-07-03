# goicy - AAC/AACplus/AACplusV2 & MP1/MP2/MP3 Icecast/Shoutcast source client

![Screenshots](http://i.imgur.com/83kEcKO.png?1)

## What's the point?
goicy is a small, portable and fast MPEG1/2/2.5 Layer1/2/3 and
AAC/AACplus/AACplusV2 Icecast/Shoutcast source client written in Go.
It is written to be extremely light-weight, cross-platform and easy to use.
It is a complete rewrite in Go of my old source client called hz and
written in Free Pascal: https://github.com/stunndard/hz


## How it works?
goicy can work in two modes: `ffmpeg` and `file`.
In `ffmpeg` mode goicy feeds audio files to ffmpeg which recodes them in realtime
to AAC or MP3 format and sends the output to an Icecast or Shoutcast server.

In `file` mode goicy reads and parses AAC or MPEG (MP1, MP2, MP3) files and sends them to
the server without any further processing.

## What files are supported?
In `ffmpeg` mode: any format of file recognizable by ffmpeg is supported.

In `file` mode: AAC/AACplus/AACplusV2 and MPEG1/MPEG2/MPEG2.5 LayerI/II/III files
can be streamed to a Icecast or Shoutcast server. All possible bitrates are
fully supported, including CBR and VBR.

## Tell me more.
 - Any audio files readable by ffmpeg are supported. All possible bitrates and their variations, including VBR.
 - Pretty precise timing.
 - Icecast and Shoutcast servers are fully supported.
 - Metadata updating supported. The metadata is read from ID3v1 and ID3v2 tags.
   It can also be read from cuesheets (.cue file with the same name as audio file).


## What platforms are supported?
Linux and Windows at the moment.

## What is required?
ffmpeg configured with `--enable-libfdk-aac`. Compile your own, or get the static compiled binaries,
for example here: https://sourceforge.net/projects/ffmpeg-hi/

## How do I install goicy?
The `go get` command will automatically fetch all dependencies required, compile the binary and place it in your $GOPATH/bin directory.

    go get github.com/stunndard/goicy

## How do I configure it?
Read `goicy.ini`. Tune it for your needs.
```INI
[stream]

; stream type
; must be 'file' or 'ffmpeg'
streamtype = ffmpeg
...

[ffmpeg]

; path to the ffmpeg executable
; can be just ffmpeg or ffmpeg.exe if ffmpeg is in PATH
; your ffmpeg should be compiled with libmp3lame and fdk_aac support enabled!
ffmpeg = ffmpeg-hi10-heaac

; sample rate in Hz
; ffmpeg will use its internal resampler
samplerate = 44100

; number of channels
; 1 = mono, 2 stereo
channels = 2

; AAC stream bitrate
bitrate = 192000

; AAC profile
; must be 'lc' for AAC Low Complexity (LC)
; 'he' for AAC SBR (High Efficiency AAC, HEAAC, AAC+, AACplus)
; 'hev2' for AAC SBR + PS (AACplusV2)
aacprofile = lc
```

Prepare your static playlist file, like:
```
/home/goicy/tracks/track1.mp3
/home/goicy/tracks/track2.flac
/home/goicy/tracks/track3.m4a
/home/goicy/tracks/track4.aac
/home/goicy/tracks/track5.ogg
```
Mixing different formats in one playlist is perfectly valid in `ffmpeg` mode!

How about a remote file, or a radio station?
```
http://your.server/your/music.mp3
https://your.box/your/music.mp4
http://your.icecast.radio/
```
Any input recognizable by ffmpeg is perfectly valid in `ffmpeg` mode
So you can easily reencode and relay internet radios too.


In `file` mode, though, you can only use AAC or MP1/MP2/MP3 files:
```
/home/goicy/tracks/track1.aac
/home/goicy/tracks/track2.aac
/home/goicy/tracks/track3.aac
/home/goicy/tracks/track4.aac
/home/goicy/tracks/track5.aac
```

or 
```
/home/goicy/tracks/track1.mp3
/home/goicy/tracks/track2.mp3
/home/goicy/tracks/track3.mp3
/home/goicy/tracks/track4.mp3
/home/goicy/tracks/track5.mp3
```

All files should be the same format, bitrate, samplerate and number of channels.
Don't mix different format (MPx/AACx) or different samplerate in one playlist if goicy is set to `file`
mode.


## How do I run it?
goicy inifile, i.e.:

    ./goicy /etc/goicy/rock.ini

