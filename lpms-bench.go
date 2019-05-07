package main

import (
	"bufio"
	"fmt"
	"os"
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/ericxtang/m3u8"
	"github.com/livepeer/lpms/ffmpeg"
)

func main() {
	if len(os.Args) <= 3 {
		panic("Expected input file, # concurrents and # segments")
	}
	fname := os.Args[1]
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
	conc, err := strconv.Atoi(os.Args[2])
	if err != nil {
		panic(err)
	}
	segs, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}

	ffmpeg.InitFFmpeg()
	var wg sync.WaitGroup
	dir := path.Dir(fname)
	start := time.Now()
	fmt.Printf("Source %s segments %d concurrency %d\n", fname, segs, conc)
	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func(wg *sync.WaitGroup) {
			for j, v := range pl.Segments {
				if j >= segs {
					break
				}
				if v == nil {
					continue
				}
				u := path.Join(dir, v.URI)
				in := &ffmpeg.TranscodeOptionsIn{
					Fname: u,
				}
				out := []ffmpeg.TranscodeOptions{
					ffmpeg.TranscodeOptions{
						Oname:   fmt.Sprintf("out/%d_240p_%d.ts", i, j),
						Profile: ffmpeg.P240p30fps16x9,
					},
				}
				t := time.Now()
				err := ffmpeg.Transcode2(in, out)
				fmt.Printf("%0.2v - %v\n", time.Now().Sub(t).Seconds(), v.URI)
				if err != nil {
					panic(err)
				}
			}
			wg.Done()
		}(&wg)
	}
	wg.Wait()
	fmt.Printf("Took %v to transcode %v segments",
		time.Now().Sub(start).Seconds(), segs)
}
