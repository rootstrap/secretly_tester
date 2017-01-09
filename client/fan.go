package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

type fanInfo struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type fan struct {
	Email, Username, Password string
	Info                      fanInfo
}

func NewFan(email, username, password string) fan {
	newFan := fan{}
	newFan.Email = email
	newFan.Username = username
	newFan.Password = password
	return newFan
}

func (this *fan) postRequest(requestUrl, jsonBodyStr string) (int, []byte) {
	var jsonStr = []byte(jsonBodyStr)
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func (this *fan) postRequestWithFanInfoResponse(requestUrl, jsonBodyStr string) (int, fanInfo) {
	status, body := this.postRequest(requestUrl, jsonBodyStr)
	fanInfoResp := fanInfo{}
	if status == 200 {
		json.Unmarshal(body, &fanInfoResp)
	}
	return status, fanInfoResp
}

func (this *fan) getRequestWithAuthorizationToken(requestUrl, token string) (int, []byte) {
	req, err := http.NewRequest("GET", requestUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FAN-TOKEN", token)
	req.Header.Set("Accept", "*/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	return resp.StatusCode, body
}

func (this *fan) getRequestWithStreamingInfoResponse(requestUrl, token string) int {
	status, body := this.getRequestWithAuthorizationToken(requestUrl, token)
	if status == 200 {
		fmt.Println(body)
		// TODO: retrieve useful infos from the response
	}
	return status
}

func (this *fan) deleteRequestWithAuthorizationToken(requestUrl, token string) int {
	req, err := http.NewRequest("POST", requestUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-FAN-TOKEN", token)
	req.Header.Set("Accept", "*/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

func (this *fan) SignUp() int {
	signupUrl := "/api/v1/fans"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","username":"` + this.Username + `","password":"` + this.Password + `","password_confirmation":"` + this.Password + `"}}`
	status, info := this.postRequestWithFanInfoResponse(urlBase+signupUrl, jsonBodyStr)
	this.Info = info
	return status
}

func (this *fan) SignIn() int {
	signinUrl := "/api/v1/fans/sign_in"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","password":"` + this.Password + `"}}`
	status, info := this.postRequestWithFanInfoResponse(urlBase+signinUrl, jsonBodyStr)
	this.Info = info
	return status
}

func (this *fan) SignInOrUpWithInstagram(instagramToken string) int {
	signInOrUpWithInstagramUrl := "/api/v1/fans/instagram_sign_in_or_up"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","oauth_token":"` + instagramToken + `"}}`
	status, info := this.postRequestWithFanInfoResponse(urlBase+signInOrUpWithInstagramUrl, jsonBodyStr)
	this.Info = info
	return status
}

func (this *fan) EnterInfluencerStream(influencerId int) int {
	enterInfluencerStreamUrl := "/api/v1/influencers/" + strconv.Itoa(influencerId) + "/streamings"
	status := this.getRequestWithStreamingInfoResponse(urlBase+enterInfluencerStreamUrl, this.Info.Token)
	return status
}

func (this *fan) LeaveInfluencerStream(influencerId int) int {
	leaveInfluencerStreamUrl := "/api/v1/influencers/" + strconv.Itoa(influencerId) + "/watchers"
	status := this.deleteRequestWithAuthorizationToken(urlBase+leaveInfluencerStreamUrl, this.Info.Token)
	return status
}
