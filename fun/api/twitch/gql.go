package api_twitch

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/0supa/supa8/config"
	"github.com/0supa/supa8/fun/api"
	"github.com/0supa/supa8/fun/utils"
)

type TwitchGQLPayload struct {
	OperationName *string `json:"operationName"`
	Query         string  `json:"query"`
	Variables     any     `json:"variables"`
}

type TwitchGQLBaseResponse struct {
	Extensions struct {
		Duration      json.Number `json:"durationMilliseconds"`
		OperationName string      `json:"operationName"`
		RequestID     string      `json:"requestID"`
	} `json:"extensions"`
}

type TwitchUserResponse struct {
	*TwitchGQLBaseResponse
	Data struct {
		User TwitchUser `json:"user"`
	} `json:"data"`
}

type TwitchUser struct {
	ID          string `json:"id,omitempty"`
	Login       string `json:"login,omitempty"`
	DisplayName string `json:"displayName,omitempty"`

	BlockedUsers *[]TwitchUser `json:"blockedUsers"`
}

type Input struct {
	ChannelID string `json:"channelID"`
	Message   string `json:"message"`
	ParentID  string `json:"replyParentMessageID"`
}

type TwitchMsg struct {
	Input `json:"input"`
}

func GetUser(login string, id string) (user TwitchUser, err error) {
	login = strings.TrimPrefix(login, "@")

	response := TwitchUserResponse{}

	payload, err := json.Marshal(TwitchGQLPayload{
		OperationName: utils.StringPtr("User"),
		Query:         "query User($login:String $id:ID) { user(lookupType:ALL login:$login id:$id) { id login displayName } }",
		Variables: TwitchUser{
			Login: login,
			ID:    id,
		},
	})
	if err != nil {
		return
	}

	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", bytes.NewBuffer(payload))
	req.Header.Set("User-Agent", api.GenericUserAgent)
	req.Header.Set("Client-Id", config.Auth.Twitch.GQL.ClientID)

	res, err := api.Generic.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return
	}

	user = response.Data.User
	return
}

func GetOwner() (user TwitchUser, err error) {
	response := TwitchUserResponse{}

	payload, err := json.Marshal(TwitchGQLPayload{
		Query: "{ user: currentUser { id login displayName blockedUsers { id login } } }",
	})
	if err != nil {
		return
	}

	req, _ := http.NewRequest("POST", "https://gql.twitch.tv/gql", bytes.NewBuffer(payload))
	req.Header.Set("User-Agent", api.GenericUserAgent)
	req.Header.Set("Client-Id", config.Auth.Twitch.GQL.ClientID)
	req.Header.Set("Authorization", "OAuth "+config.Auth.Twitch.GQL.OwnerToken)

	res, err := api.Generic.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return
	}

	user = response.Data.User
	return
}
