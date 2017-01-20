package client

import (
	"net/http"
	"strconv"
)

type FanClient struct {
	HTTPClient     http.Client
	BaseURL        string
	StreamsBaseUrl string
	StreamsToken   string
}

func NewFanClient() *FanClient {
	return &FanClient{
		BaseURL:        urlBase,
		StreamsBaseUrl: streamsUrlBase,
		StreamsToken:   streamsToken,
	}
}

type FanResponse struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

func (client *FanClient) SignUp(email, username, password string) (*FanResponse, error) {
	req := map[string]map[string]string{
		"fan": map[string]string{
			"email":                 email,
			"username":              username,
			"password":              password,
			"password_confirmation": password,
		},
	}
	var fanResponse FanResponse
	_, err := doJSONBodyRequestWithJSONResponse(client.HTTPClient, "POST", client.BaseURL+"/api/v1/fans", req, &fanResponse, map[string]string{})
	return &fanResponse, err
}

func (client *FanClient) SignIn(email, password string) (*FanResponse, error) {
	req := map[string]map[string]string{
		"fan": map[string]string{
			"email":    email,
			"password": password,
		},
	}
	var fanResponse FanResponse
	_, err := doJSONBodyRequestWithJSONResponse(client.HTTPClient, "POST", client.BaseURL+"/api/v1/fans/sign_in", req, &fanResponse, map[string]string{})
	return &fanResponse, err
}

func (client *FanClient) FollowInfluencer(token string, influencerID int) error {
	req := map[string]int{"influencer_id": influencerID}
	headers := map[string]string{"x-fan-token": token}
	resp, err := doJSONBodyRequest(client.HTTPClient, "POST", client.BaseURL+"/api/v1/fan_influencers", req, headers)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	return nil
}

func (client *FanClient) UnfollowInfluencer(token string, influencerID int) error {
	url := client.BaseURL + "/api/v1/influencers/" + strconv.Itoa(influencerID) + "/fan_influencer"
	req, err := http.NewRequest("DELETE", url, nil)
	req.Header.Set("x-fan-token", token)
	req.Header.Set("Accept", "*/*")
	resp, err := client.HTTPClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newError(req, resp, nil)
	}
	return nil
}

type JoinStreamResponse struct {
	OriginIP string `json:"originIp"`
}

func (client *FanClient) JoinStream(influencerID int, fanID int) (*JoinStreamResponse, error) {
	url := client.StreamsBaseUrl + "/streams/" + strconv.Itoa(influencerID) + "/watchers"
	body := map[string]int{"id": fanID}
	headers := map[string]string{"key": client.StreamsToken}
	var resBody JoinStreamResponse
	_, err := doJSONBodyRequestWithJSONResponse(client.HTTPClient, "POST", url, body, &resBody, headers)
	return &resBody, err
}

func (client *FanClient) LeaveStream(influencerID int, fanID int) error {
	url := client.StreamsBaseUrl + "/streams/" + strconv.Itoa(influencerID) + "/watchers/" + strconv.Itoa(fanID)
	req, err := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Key", client.StreamsToken)
	req.Header.Set("Accept", "*/*")
	resp, err := client.HTTPClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newError(req, resp, nil)
	}
	return nil
}
