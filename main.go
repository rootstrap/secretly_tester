package main

import (
	"encoding/csv"
	"flag"
	"log"
	"os"
	"strconv"
	"sync"
	"syscall"
	"time"

	"bitbucket.org/msolutionio/talkative_stream_test/client"
	"bitbucket.org/msolutionio/talkative_stream_test/rtmp"
)

func main() {
	var concurrentUsers int
	var rampUpTime time.Duration
	var existingUserOffset int
	var forceNewUsers bool
	commandName := "runtest"
	if len(os.Args) >= 2 && len(os.Args[1]) > 0 && os.Args[1][0] != '-' {
		commandName = os.Args[1]
	}
	f := flag.NewFlagSet(os.Args[0]+" "+commandName, flag.ContinueOnError)
	switch commandName {
	case "runtest":
		f.IntVar(&concurrentUsers, "users", 10, "number of concurrent users")
		f.IntVar(&existingUserOffset, "existingoffset", 0, "sequence number offset for existing users")
		f.BoolVar(&forceNewUsers, "forcenew", false, "always create new users")
		f.DurationVar(&rampUpTime, "ramp", 500*time.Millisecond, "time between users joining, e.g. 200ms")
		break
	default:
		os.Stderr.WriteString("Unknown command " + commandName + "\nshould be runtest\n")
		os.Exit(2)
	}

	err := f.Parse(os.Args[2:])
	if err != nil {
		os.Exit(2)
	}

	bumpNoFiles(8192)
	runTest(concurrentUsers, rampUpTime, existingUserOffset, forceNewUsers)
}

func runTest(concurrentUsers int, rampUpTime time.Duration, existingUserOffset int, forceNewUsers bool) {
	fanClient := client.NewFanClient()

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

	influencer, err := getInfluencer()
	if err != nil {
		log.Fatal(err)
	}

	originUrl := client.GetOriginUrl(influencer.ServerStatus.OriginIP, influencer.Username)
	log.Println("Pushing to", originUrl)
	go rtmp.NewRTMPPusher(originUrl, "640.flv").Run()

	log.Println("Waiting for 3 seconds")
	time.Sleep(3 * time.Second)

	runN(concurrentUsers, rampUpTime, func(i int) {
		startTime := time.Now()

		var fanUsername string
		if forceNewUsers {
			fanUsername = "testfan" + client.RandomString(12)
		} else {
			fanUsername = "testfan" + strconv.Itoa(existingUserOffset+i)
		}
		log.Println("Starting client", fanUsername)

		fanRes, err := fanClient.SignIn(fanUsername+"@e.com", "Password42")
		if err != nil {
			log.Println("Fan", fanUsername, "signin failure", err)
			return
		}
		if fanRes == nil {
			fanRes, err = fanClient.SignUp(fanUsername+"@e.com", fanUsername, "Password42")
			if err != nil {
				log.Println("Fan", fanUsername, "signup failure", err)
				return
			}
		} else {
			log.Println("Fan", fanUsername, "signed in")
		}
		if err = fanClient.FollowInfluencer(fanRes.Token, influencer.ID); err != nil {
			log.Println("Fan", fanUsername, "signup failure", err)
			return
		}
		log.Println("Fan", fanUsername, "followed influencer")
		joined, err := fanClient.JoinStream(influencer.ID, fanRes.ID)
		if err != nil {
			log.Println("Fan", fanUsername, "join failure", err)
			return
		}
		log.Println("Fan", fanUsername, "joined stream")
		if err = fanClient.LeaveStream(influencer.ID, fanRes.ID); err != nil {
			log.Println("Fan", fanUsername, "leave failure", err)
			return
		}
		log.Println("Fan", fanUsername, "left stream")

		rtmpUrl, err := client.GetEdgeUrl(joined.OriginIP, influencer.Username)
		if err != nil {
			log.Println(err)
			return
		}

		log.Println("Connecting client", fanUsername, "to", rtmpUrl)
		test := rtmp.NewRTMPTest(rtmpUrl)
		go func() {
			for prog := range test.Progress {
				out <- []string{
					fanUsername,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressPercent",
					strconv.FormatFloat(float64(prog.Percent), 'f', 2, 32)}
				out <- []string{
					fanUsername,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressKiloBytes",
					strconv.FormatFloat(float64(prog.KiloBytes), 'f', 2, 32)}
				out <- []string{
					fanUsername,
					strconv.FormatFloat(secsSince(startTime), 'f', 2, 32),
					"StreamProgressSeconds",
					strconv.FormatFloat(float64(prog.Seconds), 'f', 2, 32)}
			}
		}()
		err = test.Run()
		if err != nil {
			log.Println(err)
		}
	})
}

func getInfluencer() (inf *client.InfluencerResponse, err error) {
	influencerClient := client.NewInfluencerClient()

	log.Println("SignIn as influencer", "hrant@msolution.io")
	inf, err = influencerClient.InstagramSignInOrUp("hrant@msolution.io", "4352915049.1677ed0.13fb746250c84b928b37360fba9e4d57")
	if err != nil {
		return
	}

	log.Println("Creating stream status")
	if err = influencerClient.CreateStream(inf.ID, inf.Token); err != nil {
		return
	}

	log.Println("Creating stream alerts")
	if err = influencerClient.CreateStreamAlerts(inf.ID, inf.Token); err != nil {
		return
	}

	for {
		log.Println("Polling influencer for readiness")
		inf, err = influencerClient.Get(inf.ID, inf.Token)
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

func runN(count int, rampUpTime time.Duration, body func(int)) {
	var wait sync.WaitGroup
	queue := make(chan struct{}, count)
	i := 0
	for {
		time.Sleep(rampUpTime)
		wait.Add(1)
		queue <- struct{}{}
		go func() {
			defer func() {
				wait.Done()
				<-queue
			}()
			body(i)
		}()
		i++
	}
	wait.Wait()
}

func secsSince(t time.Time) float64 {
	return float64(time.Since(t)/time.Millisecond) / 1000
}

func bumpNoFiles(noFiles uint64) error {
	var rlim syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rlim)
	if err != nil {
		return err
	}
	if rlim.Cur < noFiles {
		rlim.Cur = noFiles
	}
	return syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rlim)
}
