package client

import (
	"net/http"
	"strconv"
	"time"
)

type FanClient struct {
	HTTPClient     http.Client
	BaseURL        string
	StreamsBaseUrl string
	StreamsToken   string
}

func NewFanClient(duration time.Duration) *FanClient {
	return &FanClient{
		HTTPClient: http.Client{
			Timeout: duration,
		},
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
	if err != nil {
		return nil, err
	}
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
	if err != nil {
		return nil, err
	}
	return &fanResponse, err
}

func (client *FanClient) FollowInfluencer(token string, influencerID int) error {
	req := map[string]int{"influencer_id": influencerID}
	headers := map[string]string{"x-fan-token": token}
	resp, err := doJSONBodyRequest(client.HTTPClient, "POST", client.BaseURL+"/api/v1/fan_influencers", req, headers)
	defer tryCloseRespBody(resp)
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
	defer tryCloseRespBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newError(req, resp, nil)
	}
	return nil
}

func (client *FanClient) UseCode(token string) error {
	req := map[string]int{}
	headers := map[string]string{"x-fan-token": token}
	resp, err := doJSONBodyRequest(client.HTTPClient, "POST", client.BaseURL+"/api/v1/codes/use?code_text=leti", req, headers)
	if err != nil {
		return err
	}
	defer tryCloseRespBody(resp)
	return nil
}

type GeneralMaketPlaceResponse struct {
	Influencers []struct {
		ID int `json:"id"`
	} `json:"influencers"`
}

func (client *FanClient) GetGeneralMarketplace(token string) (*GeneralMaketPlaceResponse, error) {
	url := client.BaseURL + "/api/v1/influencers/general?page=1"
	body := map[string]int{}
	headers := map[string]string{"x-fan-token": token}
	var resBody GeneralMaketPlaceResponse
	_, err := doJSONBodyRequestWithJSONResponse(client.HTTPClient, "GET", url, body, &resBody, headers)
	return &resBody, err
}

func (client *FanClient) RelationMarketplace(token string, ids []int) error {
	req := map[string][]int{"influencers": ids}
	headers := map[string]string{"x-fan-token": token}
	resp, err := doJSONBodyRequest(client.HTTPClient, "POST", client.BaseURL+"/api/v1/influencers/relation", req, headers)
	defer tryCloseRespBody(resp)
	if err != nil {
		return err
	}
	return nil
}

type JoinStreamResponse struct {
	OriginIP           string `json:"originIp"`
	InfluencerUsername string `json:"influencerUsername"`
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
	defer tryCloseRespBody(resp)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newError(req, resp, nil)
	}
	return nil
}
