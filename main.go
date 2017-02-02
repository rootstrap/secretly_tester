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

	"fmt"

	"github.com/toptier/secretly_tester/client"
	"github.com/toptier/secretly_tester/remote"
	"github.com/toptier/secretly_tester/rtmp"
	"github.com/toptier/secretly_tester/usergenerator"
)

func main() {
	bumpNoFiles(8192)
	var concurrentUsers int
	var rampUpTime time.Duration
	var existingUserOffset int
	var percentNewUsers int
	var sshHosts string
	var sshPKPath string
	var influencerID int
	var influencerEmail string
	var influencerToken string
	var testVideoPath string
	var precreateFans bool
	commandName := "runtest"
	args := os.Args[1:]
	if len(os.Args) >= 2 && len(os.Args[1]) > 0 && os.Args[1][0] != '-' {
		commandName = os.Args[1]
		args = os.Args[2:]
	}
	f := flag.NewFlagSet(os.Args[0]+" "+commandName, flag.ContinueOnError)
	f.IntVar(&concurrentUsers, "users", 10, "number of concurrent users")
	f.IntVar(&existingUserOffset, "existingoffset", 0, "sequence number offset for existing users")
	f.IntVar(&percentNewUsers, "percentnew", 0, "0-100 percentage of new fan users in test")
	f.DurationVar(&rampUpTime, "ramp", 500*time.Millisecond, "time between users joining, e.g. 200ms")
	switch commandName {
	case "runtest":
		f.StringVar(&sshHosts, "sshhosts", "", "space separated list of user@host to run test on (clustered)")
		f.StringVar(&sshPKPath, "sshkeyfile", "", "path to SSH private key file")
		f.StringVar(&influencerEmail, "email", "hrant@msolution.io", "influencer email")
		f.StringVar(&influencerToken, "token", "4352915049.1677ed0.13fb746250c84b928b37360fba9e4d57", "influencer token")
		f.StringVar(&testVideoPath, "videopath", "640.flv", "path to video file used in test")
		f.BoolVar(&precreateFans, "precreatefans", false, "pre-create fans and follow influencer (not needed on repeat runs)")
		break
	case "runfans":
		f.IntVar(&influencerID, "influencerid", 0, "influencer id to have fans join")
		break
	default:
		os.Stderr.WriteString("Unknown command " + commandName + "\nOptions:\n")
		os.Stderr.WriteString(" runtest (default)\trun full test, possibly remotely\n")
		os.Stderr.WriteString(" runfans        \tonly runs the fan portion, used by remote\n")
		os.Exit(2)
	}

	err := f.Parse(args)
	if err != nil {
		os.Exit(2)
	}

	userGenerator, err = usergenerator.NewUserGenerator(int32(existingUserOffset), int32(percentNewUsers))
	if err != nil {
		log.Fatal(err)
	}

	switch commandName {
	case "runtest":
		if sshHosts == "" {
			runInfluencer(influencerEmail, influencerToken, testVideoPath, precreateFans, concurrentUsers, percentNewUsers, func(influencerID int) {
				runFans(concurrentUsers, rampUpTime, existingUserOffset, influencerID, csvWriter())
			})
		} else {
			remote, err := remote.NewRemote(sshHosts, sshPKPath)
			if err != nil {
				log.Fatal(err)
			}
			log.Println("Connecting to remotes ", sshHosts)
			err = remote.Connect()
			if err != nil {
				log.Fatal(err)
			}
			runInfluencer(influencerEmail, influencerToken, testVideoPath, precreateFans, concurrentUsers, percentNewUsers, func(influencerID int) {
				concurrentUsersPerNode := concurrentUsers / len(remote.Nodes)
				rampUpTimePerNode := rampUpTime * time.Duration(len(remote.Nodes))
				commandString := "talkative_stream_test runfans"
				commandString += fmt.Sprintf(" -influencerid %d", influencerID)
				commandString += fmt.Sprintf(" -users %d", concurrentUsersPerNode)
				commandString += fmt.Sprintf(" -ramp %v", rampUpTimePerNode.String())
				commandString += fmt.Sprintf(" -percentnew %d", percentNewUsers)
				i := 0
				remote.StartEach(func() (string, error) {
					offset := existingUserOffset + concurrentUsers*2*i
					i++
					return commandString + fmt.Sprintf(" -existingoffset %d", offset), nil
				})
			})
		}
		break
	case "runfans":
		runFans(concurrentUsers, rampUpTime, existingUserOffset, influencerID, csvWriter())
		break
	}
}

func runInfluencer(email, token, testVideoPath string, shouldPrecreatFans bool, concurrentUsers, percentNewUsers int, run func(int)) {
	influencerCreds := signInInfluencer(email, token)

	if shouldPrecreatFans {
		precreateFans(influencerCreds.ID, concurrentUsers*(100-percentNewUsers)/100)
	}

	influencer := startInfluencer(influencerCreds)

	originURL := client.GetOriginUrl(influencer.ServerStatus.OriginIP, influencer.Username)
	log.Println("Pushing to", originURL)
	pusher := rtmp.NewRTMPPusher(originURL, testVideoPath)

	go func() {
		log.Println("Waiting 5 seconds to start fans")
		time.Sleep(5 * time.Second)
		run(influencer.ID)
	}()
	err := pusher.Run()
	if err != nil {
		log.Fatal(err)
	}
}

func precreateFans(influencerID int, nUsers int) {
	log.Printf("Pre-creating %d users and following influencer (this may take a while)\n", nUsers)
	for _, fanUsername := range userGenerator.GetExisting(nUsers) {
		fanRes, err := fanClient.SignIn(fanUsername+"@e.com", password)
		if err != nil {
			log.Println("Fan", fanUsername, "signin failure")
			fanRes, err = fanClient.SignUp(fanUsername+"@e.com", fanUsername, password)
			if err != nil {
				log.Println("Fan", fanUsername, "signup failure", err)
			}
			log.Println("Fan", fanUsername, "signed up")
		} else {
			log.Println("Fan", fanUsername, "signed in")
		}
		if err = fanClient.FollowInfluencer(fanRes.Token, influencerID); err != nil {
			log.Println("Fan", fanUsername, "follow failure", err)
		}
		log.Println("Fan", fanUsername, "followed influencer")
	}
}

func fanSignUpAndFollow(fanUsername string, influencerID int) (*client.FanResponse, error) {
	fanRes, err := fanClient.SignUp(fanUsername+"@e.com", fanUsername, password)
	if err != nil {
		log.Println("Fan", fanUsername, "signup failure", err)
		return nil, err
	}
	log.Println("Fan", fanUsername, "signed up")
	if err = fanClient.FollowInfluencer(fanRes.Token, influencerID); err != nil {
		log.Println("Fan", fanUsername, "signup failure", err)
	}
	log.Println("Fan", fanUsername, "followed influencer")
	return fanRes, err
}

func runFans(concurrentUsers int, rampUpTime time.Duration, existingUserOffset int, influencerID int, out chan []string) {
	runN(concurrentUsers, rampUpTime, func(_ int) {
		startTime := time.Now()

		fanUsername, newUser := userGenerator.Gen()
		log.Println("Starting client", fanUsername)

		var fanRes *client.FanResponse
		var err error
		if newUser {
			fanRes, err = fanSignUpAndFollow(fanUsername, influencerID)
			if err != nil {
				return
			}
		} else {
			fanRes, err = fanClient.SignIn(fanUsername+"@e.com", password)
			if err != nil {
				log.Println("Fan", fanUsername, "signin failure")
				fanRes, err = fanSignUpAndFollow(fanUsername, influencerID)
				if err != nil {
					return
				}
			} else {
				log.Println("Fan", fanUsername, "signed in")
			}
		}

		joined, err := fanClient.JoinStream(influencerID, fanRes.ID)
		if err != nil {
			log.Println("Fan", fanUsername, "join failure", err)
			return
		}
		log.Println("Fan", fanUsername, "joined stream")
		if err = fanClient.LeaveStream(influencerID, fanRes.ID); err != nil {
			log.Println("Fan", fanUsername, "leave failure", err)
			return
		}
		log.Println("Fan", fanUsername, "left stream")

		rtmpUrl, err := client.GetEdgeUrl(joined.OriginIP, joined.InfluencerUsername)
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

func signInInfluencer(email, token string) *client.InfluencerResponse {
	log.Println("SignIn as influencer", email)
	infCreds, err := influencerClient.InstagramSignInOrUp(email, token)
	if err != nil {
		log.Fatal("Failed to sign in", err)
	}
	return infCreds
}

func startInfluencer(infCreds *client.InfluencerResponse) (inf *client.InfluencerResponse) {
	var err error
	log.Println("Creating stream alerts")
	if err = influencerClient.CreateStreamAlerts(infCreds.ID, infCreds.Token); err != nil {
		log.Fatal(err)
	}

	for {
		log.Println("Polling influencer for readiness")
		inf, err = influencerClient.Get(infCreds.ID, infCreds.Token)
		if err != nil {
			log.Fatal(err)
		}
		if inf.ServerStatus.Ready {
			log.Println("Influencer ready")
			break
		}
		time.Sleep(5 * time.Second)
	}

	log.Println("Creating stream status")
	if err := influencerClient.CreateStream(infCreds.ID, infCreds.Token); err != nil {
		log.Fatal(err)
	}
	return
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

func csvWriter() chan []string {
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
	return out
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

var fanClient = client.NewFanClient()

var influencerClient = client.NewInfluencerClient()

var userGenerator *usergenerator.UserGenerator

var password = "Password42"
