package fun

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	. "github.com/0supa/supa8/fun"
	"github.com/0supa/supa8/fun/api"
	. "github.com/0supa/supa8/fun/api/twitch"
	"github.com/gempir/go-twitch-irc/v4"
)

type clip struct {
	Path    string `json:"path"`
	Error   int    `json:"error"`
	Message string `json:"message"`
}

func init() {
	Fun.Register(&Cmd{
		Name: "clip",
		Handler: func(m twitch.PrivateMessage) (err error) {
			args := strings.Split(m.Message, " ")
			if args[0] != "`clip" && args[0] != "?clip" && args[0] != "`klip" {
				return
			}

			var channel = m.Channel
			if len(args) >= 2 {
				channel = args[1]
			}

			var platform string
			if args[0] == "`klip" {
				platform = "kick"
			} else {
				platform = "twitch"
			}

			req, _ := http.NewRequest("POST", "http://127.0.0.1:8989/clip/"+
				url.PathEscape(channel)+
				fmt.Sprintf("?creator_id=%s&parent_id=%s&platform=%s", m.User.ID, m.RoomID, platform), nil)
			res, err := api.Generic.Do(req)
			if err != nil {
				return
			}

			decoder := json.NewDecoder(res.Body)
			var c clip
			err = decoder.Decode(&c)
			if err != nil {
				return
			}

			if c.Error != 0 {
				if c.Message != "" {
					_, err = Say(m.RoomID, c.Message, m.ID)
					return
				}

				_, err = Say(m.RoomID, fmt.Sprintf("something went wrong (%v)", c.Error), m.ID)
				return
			}

			_, err = Say(m.RoomID, "https://fi.supa.sh/clips/"+c.Path, m.ID)
			return
		},
	})
}
