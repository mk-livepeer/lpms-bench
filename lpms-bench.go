package main

import (
    "bufio"
    "fmt"
    "os"
    "path"
    "time"

    "github.com/livepeer/lpms/ffmpeg"
    "github.com/ericxtang/m3u8"
)

func main() {
    if len(os.Args) <= 2 {
        panic("Expected input file and out path")
    }
    fname := os.Args[1]
    wd := os.Args[2]
    f, err := os.Open(fname)
    if err != nil {
        panic(err)
    }
    p, _, err := m3u8.DecodeFrom(bufio.NewReader(f), true)
    if err != nil {
        panic(err)
    }
    pl, ok := p.(*m3u8.MediaPlaylist)
    if !ok {
        panic("Expecting media PL")
    }

    dir := path.Dir(fname)
    ffmpeg.InitFFmpeg()
    for _, v := range pl.Segments {
        if v == nil {
            continue
        }
        u := path.Join(dir, v.URI)
        t := time.Now()
        err := ffmpeg.Transcode(u, wd, []ffmpeg.VideoProfile{
            ffmpeg.P720p30fps16x9, ffmpeg.P360p30fps16x9, ffmpeg.P240p30fps16x9,
        })
        fmt.Printf("%0.2v - %v\n", time.Now().Sub(t).Seconds(), v.URI)
        if err != nil {
            panic(err)
        }
    }
}