package client

import (
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"strings"
)

type getEdgeUrlResponse struct {
	XMLName xml.Name `xml:"smil"`
	Meta    struct {
		Url string `xml:"base,attr"`
	} `xml:"head>meta"`
}

func GetEdgeUrl(host string, streamName string) (url string, err error) {
	resp, err := http.Get("http://" + host + ":1935/redirect/live/" + streamName)
	if err != nil {
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return
	}
	var parsed getEdgeUrlResponse
	err = xml.Unmarshal(body, &parsed)
	if err != nil {
		return
	}
	url = strings.Replace(parsed.Meta.Url, "_definst_", streamName, 1)
	return
}

func GetOriginUrl(host string, streamName string) string {
	return "rtmp://" + host + ":1935/live/" + streamName
}
