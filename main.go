package main

import (
	"bytes"
	"encoding/json"
	"io"
	logger "log"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/0supa/supa8/config"
	_ "github.com/0supa/supa8/fun"
	_ "github.com/0supa/supa8/fun/cmds"
)

type LogChannels struct {
	Channels []struct {
		Name   string `json:"name"`
		UserID string `json:"userID"`
	} `json:"channels"`
}

type LiveStream struct {
	Platform string `json:"platform"`
	Viewers  int    `json:"viewers"`
	Channel  struct {
		ID    string `json:"id"`
		Login string `json:"login"`
	} `json:"channel"`
}

type LivePage struct {
	Streams    []LiveStream `json:"streams"`
	NextOffset *int         `json:"nextOffset"`
}

func fetchLiveStreams() ([]LiveStream, error) {
	streams := []LiveStream{}
	offset := 0
	for {
		res, err := httpClient.Get("https://api.lurkology.com/v1/live?language=ro&offset=" + strconv.Itoa(offset) + "&limit=48")
		if err != nil {
			return nil, err
		}

		page := LivePage{}
		err = json.NewDecoder(res.Body).Decode(&page)
		res.Body.Close()
		if err != nil {
			return nil, err
		}

		streams = append(streams, page.Streams...)
		if page.NextOffset == nil {
			return streams, nil
		}
		offset = *page.NextOffset
	}
}

type JoinPayload struct {
	Channels []string `json:"channels"`
}

var wg = &sync.WaitGroup{}

var httpClient = http.Client{Timeout: time.Minute}

var log = logger.New(os.Stdout, "System ", logger.LstdFlags)

func main() {
	wg.Add(1)

	go func() {
		for range time.Tick(time.Minute * 10) {
			res, err := httpClient.Get("https://logs.supa.codes/channels")
			if err != nil {
				log.Println(err)
				continue
			}

			rustlog := LogChannels{}
			err = json.NewDecoder(res.Body).Decode(&rustlog)
			if err != nil {
				log.Println(err)
				continue
			}
			res.Body.Close()

			ignored := []string{}
			for _, ch := range rustlog.Channels {
				ignored = append(ignored, ch.UserID)
			}

			liveStreams, err := fetchLiveStreams()
			if err != nil {
				log.Println(err)
				continue
			}

			var resMsg strings.Builder
			resMsg.WriteString("logs.supa.codes -> joining new channels:")

			joinPayload := JoinPayload{}
			for _, s := range liveStreams {
				if s.Platform != "twitch" || s.Viewers < 10 || slices.Contains(ignored, s.Channel.ID) {
					continue
				}
				resMsg.WriteString(" @")
				resMsg.WriteString(s.Channel.Login)
				joinPayload.Channels = append(joinPayload.Channels, s.Channel.ID)
			}
			if len(joinPayload.Channels) == 0 {
				continue
			}

			body, err := json.Marshal(joinPayload)
			if err != nil {
				log.Println(err)
				continue
			}

			req, _ := http.NewRequest(
				"POST", "https://logs.supa.codes/admin/channels",
				bytes.NewBuffer(body),
			)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Api-Key", config.Auth.Rustlog.Key)

			res, err = httpClient.Do(req)
			if err != nil {
				log.Println(err)
				continue
			}

			if res.StatusCode != http.StatusOK {
				log.Println("failed joining new rustlog channels")

				b, err := io.ReadAll(res.Body)
				log.Println(string(b))
				if err != nil {
					log.Println(err)
				}
				continue
			}
			res.Body.Close()

			log.Println(resMsg.String())
			// fun.Say("675052240", resMsg.String(), "")
		}
	}()

	wg.Wait()
}
