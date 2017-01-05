package main

import (
	"fmt"
	"os"
	"talkative_stream_test/client"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Println("Usage: talkative_stream_test email username password")
		os.Exit(1)
	}
	fan := client.NewFan(os.Args[1], os.Args[2], os.Args[3])
	fan.PrintInfos()
	fan.SignUp()
	fmt.Println("\n--\n")
	fan.SignIn()
}
