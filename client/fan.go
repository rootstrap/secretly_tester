package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type fanInfo struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

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

func (this fan) PrintInfos() {
	fmt.Printf("'%s' '%s' '%s'\n", this.Email, this.Username, this.Password)
}

func (this fan) postRequest(requestUrl, jsonBodyStr string) (int, []byte) {
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

func (this fan) postRequestWithFanInfoResponse(requestUrl, jsonBodyStr string) (int, fanInfo) {
	status, body := this.postRequest(requestUrl, jsonBodyStr)
	fanInfoResp := fanInfo{}
	if status == 200 {
		json.Unmarshal(body, &fanInfoResp)
	}
	return status, fanInfoResp
}

func (this fan) SignUp() (int, fanInfo) {
	signupUrl := "/api/v1/fans"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","username":"` + this.Username + `","password":"` + this.Password + `","password_confirmation":"` + this.Password + `"}}`
	status, resp := this.postRequestWithFanInfoResponse(urlBase+signupUrl, jsonBodyStr)
	return status, resp
}

func (this fan) SignIn() (int, fanInfo) {
	signinUrl := "/api/v1/fans/sign_in"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","password":"` + this.Password + `"}}`
	status, resp := this.postRequestWithFanInfoResponse(urlBase+signinUrl, jsonBodyStr)
	return status, resp
}

func (this fan) SignInOrUpWithInstagram(instagramToken string) (int, fanInfo) {
	signInOrUpWithInstagramUrl := "/api/v1/fans/instagram_sign_in_or_up"
	var jsonBodyStr string = `{"fan":{"email":"` + this.Email + `","oauth_token":"` + instagramToken + `"}}`
	status, resp := this.postRequestWithFanInfoResponse(urlBase+signInOrUpWithInstagramUrl, jsonBodyStr)
	return status, resp
}
