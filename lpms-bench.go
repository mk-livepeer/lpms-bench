// Loads any file and runs it once via transcode loop. HLS output.
// (Input can be HLS as well but the outputs won't be aligned.)

package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	//"runtime/pprof"

	"github.com/livepeer/lpms/ffmpeg"
	"github.com/livepeer/m3u8"
)

func validRenditions() []string {
	valids := make([]string, len(ffmpeg.VideoProfileLookup))
	for p, _ := range ffmpeg.VideoProfileLookup {
		valids = append(valids, p)
	}
	return valids
}

func str2profs(inp string) []ffmpeg.VideoProfile {
	profs := []ffmpeg.VideoProfile{}
	strs := strings.Split(inp, ",")
	for _, k := range strs {
		p, ok := ffmpeg.VideoProfileLookup[k]
		if !ok {
			panic(fmt.Sprintf("Invalid rendition %s. Valid renditions are:\n%s", k, validRenditions()))
		}
		profs = append(profs, p)
	}
	return profs
}

func main() {
	const usage = "Expected: [input file] [output prefix] [profiles] [sw/nv] <nv-device>"
	if len(os.Args) <= 5 {
		panic(usage)
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
	_, ok := p.(*m3u8.MediaPlaylist)
	if !ok {
		panic("Expecting media PL")
	}
	pfx := os.Args[2]
	profiles := str2profs(os.Args[3])
	accelStr := os.Args[4]
	accel := ffmpeg.Software
	devices := []string{}
	if "nv" == accelStr {
		accel = ffmpeg.Nvidia
		if len(os.Args) <= 5 {
			panic(usage)
		}
		devices = strings.Split(os.Args[5], ",")
	}

	// only run the first rendition

	rendition := []ffmpeg.VideoProfile{profiles[0]}
	elapsedOne, err := benchmark(fname, fmt.Sprintf("%s/one/", pfx), accelStr, 1, rendition, accel, devices)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}
	fmt.Fprintf(os.Stderr, "Took %v to transcode 1 rendition: %v", elapsedOne.Seconds(), rendition)

	// run all renditions

	elapsedAll, err := benchmark(fname, fmt.Sprintf("%s/all/", pfx), accelStr, len(profiles), profiles, accel, devices)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		return
	}
	fmt.Fprintf(os.Stderr, "Took %v to transcode %d renditions: %v", elapsedAll.Seconds(), len(profiles), profiles)

}

func benchmark(fname, pfx, accelStr string, conc int, profiles []ffmpeg.VideoProfile, accel ffmpeg.Acceleration, devices []string) (time.Duration, error) {
	if err := os.MkdirAll(pfx, os.ModePerm); err != nil {
		return 0, err
	}
	ffmpeg.InitFFmpeg()
	var wg sync.WaitGroup
	start := time.Now()
	fmt.Fprintf(os.Stderr, "Source %s concurrency %d\n", fname, conc)
	fmt.Println("time,stream")
	for i := 0; i < conc; i++ {
		wg.Add(1)
		go func(k int, wg *sync.WaitGroup) {
			in := &ffmpeg.TranscodeOptionsIn{
				Fname: fname,
				Accel: accel,
			}
			if ffmpeg.Software != accel {
				in.Device = devices[k%len(devices)]
			}
			profs2opts := func(profs []ffmpeg.VideoProfile) []ffmpeg.TranscodeOptions {
				opts := []ffmpeg.TranscodeOptions{}
				for _, p := range profs {
					o := ffmpeg.TranscodeOptions{
						Oname:   fmt.Sprintf("%s%s_%s_%d.m3u8", pfx, accelStr, p.Name, k),
						Profile: p,
						Accel:   accel,
					}
					opts = append(opts, o)
				}
				return opts
			}
			out := profs2opts(profiles)
			t := time.Now()
			err := ffmpeg.Transcode2(in, out)
			end := time.Now()
			fmt.Printf("%s,%d,%0.2v\n", end.Format("2006-01-02 15:04:05.999999999"), k, end.Sub(t).Seconds())
			if err != nil {
				panic(err)
			}
			wg.Done()
		}(i, &wg)
	}
	wg.Wait()
	return time.Now().Sub(start), nil
}
