package client

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

type influencerInfo struct {
	Id       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

type influencer struct {
	OauthToken, Email string
	Info              influencerInfo
}

func NewInfluencer(oauthToken, email string) influencer {
	newInfluencer := influencer{}
	newInfluencer.OauthToken = oauthToken
	newInfluencer.Email = email
	return newInfluencer
}

func (this *influencer) postRequest(requestUrl, jsonBodyStr string) (int, []byte) {
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

func (this *influencer) postRequestWithAuthorizationToken(requestUrl, jsonBodyStr, token string) (int, []byte) {
	var jsonStr = []byte(jsonBodyStr)
	req, err := http.NewRequest("POST", requestUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-INFLUENCER-TOKEN", token)
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

func (this *influencer) postRequestWithInfluencerInfoResponse(requestUrl, jsonBodyStr string) (int, influencerInfo) {
	status, body := this.postRequest(requestUrl, jsonBodyStr)
	influencerInfoResp := influencerInfo{}
	if status == 200 {
		json.Unmarshal(body, &influencerInfoResp)
	}
	return status, influencerInfoResp
}

func (this *influencer) deleteRequestWithAuthorizationToken(requestUrl, token string) int {
	req, err := http.NewRequest("POST", requestUrl, nil)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-INFLUENCER-TOKEN", token)
	req.Header.Set("Accept", "*/*")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	return resp.StatusCode
}

func (this *influencer) InstagramLogin() int {
	instagramLoginUrl := "/api/v1/influencers/instagram_log_in"
	var jsonBodyStr string = `{"influencer":{"oauth_token":"` + this.OauthToken + `"}}`
	status, info := this.postRequestWithInfluencerInfoResponse(urlBase+instagramLoginUrl, jsonBodyStr)
	this.Info = info
	return status
}

func (this *influencer) InstagramSignInOrUp() int {
	instagramSignInOrUpUrl := "/api/v1/influencers/instagram_sign_in_or_up"
	var jsonBodyStr string = `{"influencer":{"oauth_token":"` + this.OauthToken + `","email":"` + this.Email + `"}}`
	status, info := this.postRequestWithInfluencerInfoResponse(urlBase+instagramSignInOrUpUrl, jsonBodyStr)
	this.Info = info
	return status
}

func (this *influencer) SignIn() int {
	status := this.InstagramLogin()
	if status != 200 {
		status = this.InstagramSignInOrUp()
	}
	return status
}

func (this *influencer) CreateStream() int {
	createStreamUrl := "/api/v1/influencers/" + strconv.Itoa(this.Info.Id) + "/streamings"
	var jsonBodyStr string = "{}"
	status, _ := this.postRequestWithAuthorizationToken(urlBase+createStreamUrl, jsonBodyStr, this.Info.Token)
	return status
}

func (this *influencer) DeleteStream() int {
	deleteStreamUrl := "/api/v1/influencers/" + strconv.Itoa(this.Info.Id) + "/streamings"
	status := this.deleteRequestWithAuthorizationToken(urlBase+deleteStreamUrl, this.Info.Token)
	return status
}
