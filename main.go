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

	influencer, _ := getInfluencer()

	originUrl := client.GetOriginUrl(influencer.ServerStatus.OriginIP, influencer.Username)
	log.Println("Pushing to", originUrl)
	go rtmp.NewRTMPPusher(originUrl, "640.flv").Run()

	log.Println("Waiting for 3 seconds")
	time.Sleep(3 * time.Second)

	runN(concurrentUsers, rampUpTime, func() {
		sessionId := client.RandomString(32)
		log.Println("Starting client", sessionId)
		startTime := time.Now()

		rtmpUrl, _ := client.GetEdgeUrl(influencer.ServerStatus.OriginIP, influencer.Username)
		// something wrong with edge
		rtmpUrl = client.GetOriginUrl(influencer.ServerStatus.OriginIP, influencer.Username)

		if false { //disable fan signup
			fanUsername := "fan" + client.RandomString(12)
			fan := client.NewFan(fanUsername+"@e.com", fanUsername, "Password42")
			status := fan.SignUp()
			log.Println("Fan signup status: ", status)
			if status == 200 {
				status = fan.EnterInfluencerStream(influencer.ID)
				log.Println("Fan enters influencer stream status: ", status)
			}
		}

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

func getInfluencer() (inf client.GetInfluencerResponse, err error) {
	influencer := client.NewInfluencer("4352915049.1677ed0.13fb746250c84b928b37360fba9e4d57", "hrant@msolution.io")
	log.Println("SignIn as influencer", "hrant@msolution.io")
	status := influencer.SignIn()

	if status == 200 {
		log.Println("Creating stream status")
		status = influencer.CreateStream()
		log.Println("Creating stream alerts")
		status = influencer.CreateStreamAlerts()

		for {
			log.Println("Polling influencer for readiness")
			inf, err = influencer.GetInfluencer()
			if err != nil {
				return
			}
			if inf.ServerStatus.Ready {
				log.Println("Influencer ready")
				return
			}
			time.Sleep(5 * time.Second)
		}
	}
	return
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
