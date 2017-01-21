package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"time"
)

const urlBase string = "http://talkative-staging.herokuapp.com" // TODO: put in a config file
const streamsUrlBase string = "http://secretly-sender.herokuapp.com"
const streamsToken string = "SENDERHQ2016"

func RandomString(strlen int) string {
	rand.Seed(time.Now().UTC().UnixNano())
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, strlen)
	for i := 0; i < strlen; i++ {
		result[i] = chars[rand.Intn(len(chars))]
	}
	return string(result)
}

func newError(req *http.Request, resp *http.Response, body []byte) error {
	s := ""
	if req != nil {
		reqBytes, _ := httputil.DumpRequest(req, false)
		s += string(reqBytes)
	}
	if body != nil {
		resBytes, _ := httputil.DumpResponse(resp, false)
		s += string(resBytes)
		s += string(body)
	} else {
		resBytes, err := httputil.DumpResponse(resp, true)
		if err != nil {
			return err
		}
		s += string(resBytes)
	}
	return errors.New(s)
}

func doReqRep(client http.Client, meth, url string, headers map[string]string) error {
	req, err := http.NewRequest(meth, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "*/*")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	defer resp.Body.Close()
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return newError(req, resp, nil)
	}
	return nil
}

func doJSONBodyRequest(client http.Client, meth, url string, reqBody interface{}, headers map[string]string) (*http.Response, error) {
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(meth, url, bytes.NewBuffer(reqBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, newError(req, resp, nil)
	}
	return resp, nil
}

func doJSONBodyRequestWithJSONResponse(client http.Client, meth, url string, reqBody, respBody interface{}, headers map[string]string) (*http.Response, error) {
	resp, err := doJSONBodyRequest(client, meth, url, reqBody, headers)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return resp, err
	}
	return resp, json.Unmarshal(respBodyBytes, respBody)
}
