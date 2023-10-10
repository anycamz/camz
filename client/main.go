package main

import (
	"fmt"

	"github.com/go-resty/resty/v2"
	"github.com/osintami/camz/base"
	"github.com/osintami/camz/sink"
	"github.com/pborman/getopt"
)

func main() {
	svr := getopt.StringLong("svr", 's', "localhost:8080", "mjpeg server addr:port")
	key := getopt.StringLong("api-key", 'a', "changeme", "mjpeg server API key")
	getopt.Parse()
	config := &base.CameraConfig{}
	sink.LoadJson("camera.json", config)

	url := fmt.Sprintf("http://%s/v1/config", *svr)
	resp, err := resty.New().R().SetHeader("Content-Type", "application/json").SetHeader("X-Api-Key", *key).SetBody(config).Post(url)
	if err != nil {
		fmt.Println("http status ", resp.StatusCode())
		fmt.Println(string(resp.Body()))
		fmt.Println(err)
		return
	}
	fmt.Println(string(resp.Body()))
}
