package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strconv"
	"sync"
	"talkative_stream_test/client"
	"talkative_stream_test/rtmp"
	"time"
)

func main() {
	var concurrentUsers int
	var rampUpTime time.Duration
	flag.IntVar(&concurrentUsers, "users", 10, "number of concurrent users")
	flag.DurationVar(&rampUpTime, "ramp", 500*time.Millisecond, "time between users joining, e.g. 200ms")
	flag.Parse()

	out := make(chan []string)
	go func() {
		csvWriter := csv.NewWriter(os.Stdout)
		for row := range out {
			err := csvWriter.Write(row)
			csvWriter.Flush()
			if err != nil {
				log.Println(err)
			}
		}
	}()

	runN(concurrentUsers, rampUpTime, func() {
		sessionId := client.RandomString(32)
		log.Println("Starting client", sessionId)
		startTime := time.Now()

		rtmpUrl := "rtmp://dev.wowza.longtailvideo.com/vod/_definst_/sintel/640.mp4"
		log.Println("Connecting client", sessionId, "to", rtmpUrl)
		test := rtmp.NewRTMPTest(rtmpUrl)
		go func() {
			for prog := range test.Progress {
				out <- []string{
					sessionId,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressPercent",
					strconv.FormatFloat(float64(prog.Percent), 'f', 2, 32)}
				out <- []string{
					sessionId,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressKiloBytes",
					strconv.FormatFloat(float64(prog.KiloBytes), 'f', 2, 32)}
				out <- []string{
					sessionId,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressSeconds",
					strconv.FormatFloat(float64(prog.Seconds), 'f', 2, 32)}
			}
		}()
		err := test.Run()
		if err != nil {
			log.Println(err)
		}
	})
}

func runN(count int, rampUpTime time.Duration, body func()) {
	var wait sync.WaitGroup
	queue := make(chan struct{}, count)
	for {
		time.Sleep(rampUpTime)
		wait.Add(1)
		queue <- struct{}{}
		go func() {
			defer func() {
				wait.Done()
				<-queue
			}()
			body()
		}()
	}
	wait.Wait()
}

func secsSince(t time.Time) float64 {
	return float64(time.Since(t)/time.Millisecond) / 1000
}
