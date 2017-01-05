package client

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
)

const urlBase string = "http://talkative-staging.herokuapp.com" // TODO: put in a config file

type fan struct {
	Email, Username, Password string
}

func NewFan(email, username, password string) fan {
	newFan := fan{}
	newFan.Email = email
	newFan.Username = username
	newFan.Password = password
	return newFan
}

func (f fan) PrintInfos() {
	fmt.Printf("'%s' '%s' '%s'\n", f.Email, f.Username, f.Password)
}

func (f fan) SignUp() {
	signupUrl := "/api/v1/fans"
	requestUrl := urlBase + signupUrl
	fmt.Println(requestUrl)

	var jsonBodyStr string = `{"fan":{"email":"` + f.Email + `","username":"` + f.Username + `","password":"` + f.Password + `","password_confirmation":"` + f.Password + `"}}`
	var jsonStr = []byte(jsonBodyStr)
	fmt.Println(jsonBodyStr)
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}

func (f fan) SignIn() {
	signupUrl := "/api/v1/fans/sign_in"
	requestUrl := urlBase + signupUrl
	fmt.Println(requestUrl)

	var jsonBodyStr string = `{"fan":{"email":"` + f.Email + `","password":"` + f.Password + `"}}`
	var jsonStr = []byte(jsonBodyStr)
	fmt.Println(jsonBodyStr)
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("response Status:", resp.Status)
	fmt.Println("response Headers:", resp.Header)
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Println("response Body:", string(body))
}
