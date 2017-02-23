package client

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
)

type InfluencerClient struct {
	HTTPClient http.Client
	BaseURL    string
}

func NewInfluencerClient() *InfluencerClient {
	return &InfluencerClient{
		BaseURL: urlBase,
	}
}

type InfluencerResponse struct {
	ID           int    `json:"id"`
	Email        string `json:"email"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	ServerStatus struct {
		OriginIP  string `json:"origin_ip"`
		Launching bool   `json:"servers_launching"`
		Ready     bool   `json:"servers_ready"`
	} `json:"servers_status"`
}

type influencer struct {
	OauthToken, Email string
	Info              InfluencerResponse
}

func (client *InfluencerClient) InstagramSignInOrUp(email, instaToken string) (*InfluencerResponse, error) {
	url := client.BaseURL + "/api/v1/influencers/instagram_sign_in_or_up"
	req := map[string]map[string]string{
		"influencer": map[string]string{
			"email":       email,
			"oauth_token": instaToken,
		},
	}
	var influencerResp InfluencerResponse
	_, err := doJSONBodyRequestWithJSONResponse(client.HTTPClient, "POST", url, req, &influencerResp, map[string]string{})
	return &influencerResp, err
}

func (client *InfluencerClient) CreateStream(influencerID int, token string) error {
	url := client.BaseURL + "/api/v1/influencers/" + strconv.Itoa(influencerID) + "/streamings"
	return doReqRep(client.HTTPClient, "POST", url, map[string]string{"x-influencer-token": token})
}

func (client *InfluencerClient) CreateStreamAlerts(influencerID int, token string) error {
	url := client.BaseURL + "/api/v1/influencers/" + strconv.Itoa(influencerID) + "/stream_alerts"
	return doReqRep(client.HTTPClient, "POST", url, map[string]string{"x-influencer-token": token})
}

func (client *InfluencerClient) DeleteStream(influencerID int, token string) error {
	url := "/api/v1/influencers/" + strconv.Itoa(influencerID) + "/streamings"
	return doReqRep(client.HTTPClient, "DELETE", url, map[string]string{"x-influencer-token": token})
}

func (client *InfluencerClient) Get(influencerID int, token string) (*InfluencerResponse, error) {
	url := client.BaseURL + "/api/v1/influencers/" + strconv.Itoa(influencerID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "*/*")
	req.Header.Set("x-influencer-token", token)
	resp, err := client.HTTPClient.Do(req)
	defer tryCloseRespBody(resp)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, newError(req, resp, nil)
	}
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var influencerResp InfluencerResponse
	err = json.Unmarshal(bodyBytes, &influencerResp)
	return &influencerResp, err
}
