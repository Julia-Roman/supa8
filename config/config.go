package config

import (
	"log"
	"os"

	"gopkg.in/yaml.v3"
)

var Meta = struct {
	Channels        []string `yaml:"channels"`
	PrivilegedUsers []string `yaml:"privileged_users"`
	BotUsers        []string `yaml:"bot_users"`
	Functions       map[string]struct {
		Channels []string `yaml:"channels"`
	} `yaml:"functions"`
}{}

var Auth = struct {
	Twitch struct {
		GQL struct {
			ClientID   string `yaml:"client_id"`
			Token      string `yaml:"token"`
			OwnerToken string `yaml:"owner_token"`
			UserID     string `yaml:"user_id"`
		} `yaml:"gql"`
		Helix struct {
			ClientID string `yaml:"client_id"`
			Token    string `yaml:"token"`
		} `yaml:"helix"`
	} `yaml:"twitch"`
	Cloudflare struct {
		Key   string `yaml:"key"`
		AccID string `yaml:"account_id"`
	} `yaml:"cloudflare"`
	OpenAI struct {
		Key string `yaml:"key"`
	} `yaml:"openai"`
	Rustlog struct {
		Key string `yaml:"key"`
	} `yaml:"rustlog"`
}{}

func loadConfig(file string, y interface{}) {
	data, err := os.ReadFile(file)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, y)
	if err != nil {
		log.Fatal(err)
	}
}

func init() {
	loadConfig("meta.yml", &Meta)
	loadConfig("auth.yml", &Auth)
}
