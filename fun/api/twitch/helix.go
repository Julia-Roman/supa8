package api_twitch

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/0supa/supa8/config"
	"github.com/0supa/supa8/fun/api"
	api_kappa "github.com/0supa/supa8/fun/api/kappa"
)

type TwitchSendMsgPayload struct {
	ChannelID     string `json:"broadcaster_id"`
	SenderID      string `json:"sender_id"`
	Message       string `json:"message"`
	ReplyParentID string `json:"reply_parent_message_id,omitempty"`
}

type TwitchSendMsgResponse struct {
	Data []struct {
		MessageID  string `json:"message_id"`
		IsSent     bool   `json:"is_sent"`
		DropReason *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"drop_reason"`
	} `json:"data"`
}

const zeroWidthChar = "\U000E0000"

func Say(channelID string, message string, parentID string, ctx ...int) (response TwitchSendMsgResponse, err error) {
	if len(ctx) == 0 {
		ctx = append(ctx, 1)
	} else {
		ctx[0] = ctx[0] + 1
	}

	og := message
	uploadMessage := func() (upload api_kappa.FileUpload) {
		rc := io.NopCloser(strings.NewReader(og))
		defer rc.Close()

		// TODO: someway handle err?
		upload, _ = api_kappa.UploadFile(rc, "msg.txt", "text/plain")
		return
	}

	if len(message) > 400 {
		message = message[:350] + "... " + uploadMessage().Link
	}

	payload, err := json.Marshal(TwitchSendMsgPayload{
		ChannelID:     channelID,
		SenderID:      config.Auth.Twitch.GQL.UserID,
		Message:       message,
		ReplyParentID: parentID,
	})
	if err != nil {
		return
	}

	req, _ := http.NewRequest("POST", "https://api.twitch.tv/helix/chat/messages", bytes.NewBuffer(payload))
	req.Header.Set("Client-Id", config.Auth.Twitch.Helix.ClientID)
	req.Header.Set("Authorization", "Bearer "+config.Auth.Twitch.Helix.Token)
	req.Header.Set("Content-Type", "application/json")

	res, err := api.Generic.Do(req)
	if err != nil {
		return
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		return
	}

	if len(response.Data) == 0 || !response.Data[0].IsSent {
		dropReason := "unknown"
		if len(response.Data) > 0 && response.Data[0].DropReason != nil {
			dropReason = response.Data[0].DropReason.Code
		}

		if ctx[0] > 3 {
			return response, fmt.Errorf("message dropped after %v attempts (%s)", ctx[0], dropReason)
		}

		if res.StatusCode == 429 {
			time.Sleep(time.Second)
			suf := " " + zeroWidthChar
			message, found := strings.CutSuffix(message, suf)
			if !found {
				message += suf
			}

			return Say(channelID, message, parentID, ctx...)
		}

		return response, fmt.Errorf("message dropped (%s): %s", dropReason, uploadMessage().Link)
		// return Say(channelID, fmt.Sprintf("(%s) failed to send reply: %s", *dropReason, uploadMessage().Link), parentID, append(ctx[:i], ctx[i]+1)...)
	}

	return
}
