package main

import (
	"fmt"
	"talkative_stream_test/client"
)

func main() {
	influencer := client.NewInfluencer("4352915049.1677ed0.13fb746250c84b928b37360fba9e4d57", "hrant@msolution.io")
	status := influencer.SignIn()
	fmt.Println("Influencer signin status: ", status)

	if status == 200 {
		status = influencer.CreateStream()
		fmt.Println("Create stream status: ", status)

		fanUsername := "fan" + client.RandomString(12)
		fan := client.NewFan(fanUsername+"@e.com", fanUsername, "Password42")
		status = fan.SignUp()
		fmt.Println("Fan signup status: ", status)
		if status == 200 {
			status = fan.EnterInfluencerStream(influencer.Info.Id)
			fmt.Println("Fan enters influencer stream status: ", status)
			status = fan.LeaveInfluencerStream(influencer.Info.Id)
			fmt.Println("Fan leaves influencer stream status: ", status)
		}

		status = influencer.DeleteStream()
		fmt.Println("Delete stream status: ", status)
	}
}
