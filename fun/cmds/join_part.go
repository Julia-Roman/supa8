package fun

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/0supa/supa8/config"
	. "github.com/0supa/supa8/fun"
	. "github.com/0supa/supa8/fun/api/twitch"
	"github.com/0supa/supa8/fun/utils"
	"github.com/gempir/go-twitch-irc/v4"
	"gopkg.in/yaml.v3"
)

func init() {
	Fun.Register(&Cmd{
		Name: "join_part",
		Handler: func(m twitch.PrivateMessage) (err error) {
			args := strings.Split(m.Message, " ")
			if !utils.IsPrivileged(m.User.ID) || len(args) < 2 || (args[0] != "`join" && args[0] != "`part") {
				return
			}

			user, err := GetUser(args[1], "")
			if err != nil {
				Say(m.RoomID, "failed getting user: "+err.Error(), m.ID)
				return
			}

			if user.ID == "" {
				Say(m.RoomID, "user not found", m.ID)
				return
			}

			v := "+"
			switch args[0] {
			case "`join":
				if slices.Contains(config.Meta.Channels, user.ID) {
					_, err = Say(m.RoomID, "already joined", m.ID)
					return
				}
				Join(user.Login)
				config.Meta.Channels = append(config.Meta.Channels, user.ID)
			case "`part":
				v = "-"
				i := slices.Index(config.Meta.Channels, user.ID)
				if i == -1 {
					_, err = Say(m.RoomID, "not joined", m.ID)
					return
				}
				Part(user.Login)
				config.Meta.Channels = slices.Delete(config.Meta.Channels, i, i+1)
			}

			out, err := yaml.Marshal(config.Meta)
			if err != nil {
				return
			}

			err = os.WriteFile("meta.yml", out, 0644)
			if err != nil {
				return
			}

			_, err = Say(m.RoomID, fmt.Sprintf("%s #%s (%s)", v, user.Login, user.ID), m.ID)
			return
		},
	})
}
