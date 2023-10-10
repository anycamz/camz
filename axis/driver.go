// Copyright © 2022 Sloan Childers
package axis

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"net/http"
	"strconv"
	"time"

	"github.com/osintami/camz/base"
	"github.com/rs/zerolog/log"
)

const (
	IP_ADDR  = "ip_addr"
	IP_PORT  = "ip_port"
	NAME     = "name"
	ID       = "id"
	USERNAME = "username"
	PASSWORD = "password"
	URI      = "uri"
	WIDTH    = "width"
	HEIGHT   = "height"
	DRIVER   = "driver"
	FORMAT   = "format"
)

// "uuid": "<uuid>",
// "plugin": "axis241q",
// "enabled": true,
// "name": "Monument",
// "width": 320,
// "height": 240,
// "rate": 20,
// "addr": "88.53.197.250",
// "port": 80,
// "user": "",
// "pass": "",
// "uri": "axis-cgi/mjpg/video.cgi",

type Axis struct {
	client *http.Client
	req    *http.Request
	resp   *http.Response
	config *base.CameraConfig
	frame  []byte
	stop   bool
	mutex  sync.Mutex
}

type Axis241Q struct {
	Axis
}

func NewDriver(config *base.CameraConfig) base.IDriver {
	return &Axis{
		config: config,
		frame:  base.EmptyFrame(config.Width, config.Height)}
}

func (x *Axis) ListFormatsAndFrameSizes() base.Formats {
	// TODO:  add formats and sizes
	return base.Formats{}
}

func (x *Axis) Grab() base.IFrame {
	x.mutex.Lock()
	defer x.mutex.Unlock()
	//data := make([]byte, len(x.frame))
	//copy(data, x.frame)
	frame := base.NewFrame(x.config)
	frame.SetImage(x.frame, base.JPEG)
	return frame
}

func (x *Axis) Stream() {
	go x.stream(nil)
}

func (x *Axis) Open() error {
	var err error
	x.req, err = http.NewRequest("GET", x.getURL(), nil)
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("http.NewRequest")
		return err
	}

	if x.config.User != "" {
		x.req.SetBasicAuth(x.config.User, x.config.Pass)
	}
	x.client = &http.Client{CheckRedirect: x.redirectPolicyFunc}
	x.resp, err = x.client.Do(x.req)
	if err != nil {
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).Msg("client.Do")
		return err
	}
	x.stop = false
	return nil
}

func (x *Axis) Stop() {
	x.mutex.Lock()
	x.stop = true
	x.frame = base.EmptyFrame(x.config.Width, x.config.Height)
	time.Sleep(1000 * time.Millisecond)
	if x.resp.Body != nil {
		x.resp.Body.Close()
	}
	x.client.CloseIdleConnections()
	x.mutex.Unlock()
}

func (x *Axis) Reset() error {
	x.Stop()
	err := x.Open()
	if err != nil {
		json, err := json.Marshal(x.config)
		log.Error().Err(err).Str("component", "driver").Str("name", x.config.Name).RawJSON("config", json).Msg("Open")
		return err
	}
	x.Stream()
	return nil
}

func (x *Axis) redirectPolicyFunc(req *http.Request, via []*http.Request) error {
	req.SetBasicAuth(x.config.User, x.config.Pass)
	return nil
}

// base Axis camera
func (x *Axis) getURL() string {
	config := x.config
	url := fmt.Sprintf("http://%s:%d/%s?resolution=%dx%d",
		config.Addr,
		config.Port,
		config.Uri,
		config.Width,
		config.Height)
	return url
}

// Axis241Q
func (x *Axis241Q) getURL() string {
	config := x.config
	return fmt.Sprintf("http://%s:%d/%s?resolution=%dx%d&camera=%s&showlength=true&textpos=top&textcolor=white&textbackgroundcolor=black&date=1&clock=1",
		config.Addr,
		config.Port,
		config.Uri,
		config.Width,
		config.Height,
		config.Name)
}

// NOTE:  this can be modified and used to restream a clip with sleeps to control frame rate
func (x *Axis) stream(writer io.Writer) {
	defer func() {
		if r := recover(); r != nil {
			log.Error().Str("component", "axis").Str("name", x.config.Name).Msg("fatal")
			x.Reset()
		}
	}()

	var readBuffer []byte
	var jpegBuffer []byte
	var jpegLength uint64
	var err error

	// streaming from a live camera
	reader := bufio.NewReader(x.resp.Body)
	defer x.resp.Body.Close()

	for {
		startTime := time.Now().UnixMilli()
		for {
			readBuffer, _, err = reader.ReadLine()
			if err != nil {
				return
			}
			if string(readBuffer) == base.BOUNDARY {
				if writer != nil {
					writer.Write(readBuffer)
				}
				break
			}
		}

		// Content-Type
		reader.Discard(base.CONTENT_TYPE_SIZE + base.EOL_SIZE)

		// Content-Length
		reader.Discard(base.CONTENT_LENGTH_SIZE)
		readBuffer, _, err = reader.ReadLine()
		if x.checkErr("contentLength", err) {
			return
		}

		// parse bytes jpeg image size
		jpegLength, err = strconv.ParseUint(string(readBuffer), 10, 32)
		if x.checkErr("jpegLength", err) {
			return
		}

		// skip empty line
		reader.Discard(base.EOL_SIZE)

		jpegBuffer = make([]byte, jpegLength)
		n, err := io.ReadFull(reader, jpegBuffer)
		if x.checkErr("readJpeg", err) {
			return
		}

		if jpegLength > 0 && jpegLength == uint64(n) {
			x.mutex.Lock()
			if x.stop {
				x.mutex.Unlock()
				return
			}
			x.frame = jpegBuffer
			x.mutex.Unlock()
			base.Sleep(int64(x.config.Rate), time.Now().UnixMilli()-startTime)
		}
	}
}

func (x *Axis) checkErr(key string, err error) bool {
	if err != nil {
		log.Error().Err(err).Str("component", "axis").Str("name", x.config.Name).Msg(key)
		return true
	}
	return false
}
