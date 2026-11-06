package fun

import (
	logger "log"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/0supa/supa8/config"
	api_twitch "github.com/0supa/supa8/fun/api/twitch"

	"github.com/gempir/go-twitch-irc/v4"
)

var Client = twitch.NewAnonymousClient()

var log = logger.New(os.Stdout, "TMI ", logger.LstdFlags)

type joinState struct {
	confirmed   bool
	lastAttempt time.Time
}

// go-twitch-irc considers a channel joined as soon as JOIN is sent, so joins
// dropped by Twitch's per-IP ratelimit are never retried unless we track the
// server's confirmation ourselves.
var joins = struct {
	sync.Mutex
	m map[string]*joinState
}{m: map[string]*joinState{}}

const joinRetryInterval = time.Minute

func Join(login string) {
	joins.Lock()
	joins.m[login] = &joinState{lastAttempt: time.Now()}
	joins.Unlock()
	Client.Join(login)
}

func Part(login string) {
	joins.Lock()
	delete(joins.m, login)
	joins.Unlock()
	Client.Depart(login)
}

func setConfirmed(login string, confirmed bool) {
	joins.Lock()
	defer joins.Unlock()
	if s, ok := joins.m[login]; ok {
		s.confirmed = confirmed
		s.lastAttempt = time.Now()
	}
}

func retryUnconfirmedJoins() {
	for range time.Tick(joinRetryInterval) {
		retry := []string{}
		joins.Lock()
		for login, s := range joins.m {
			if !s.confirmed && time.Since(s.lastAttempt) >= joinRetryInterval {
				s.lastAttempt = time.Now()
				retry = append(retry, login)
			}
		}
		joins.Unlock()

		if len(retry) == 0 {
			continue
		}
		log.Println("retrying unconfirmed joins", retry)
		for _, login := range retry {
			// Depart clears the library's joined flag; without it Join is a no-op.
			Client.Depart(login)
			Client.Join(login)
		}
	}
}

func init() {
	if config.Auth.Twitch.GQL.OwnerToken != "" {
		LoadBlocklist()
		log.Printf("%v blocked Twitch users\n", len(Fun.BlockedUserIDs))
	}

	Client.OnConnect(func() {
		log.Println("connected")

		joins.Lock()
		for _, s := range joins.m {
			s.confirmed = false
			s.lastAttempt = time.Now()
		}
		joins.Unlock()
	})

	Client.OnSelfJoinMessage(func(m twitch.UserJoinMessage) {
		log.Println("joined", m.Channel)
		setConfirmed(m.Channel, true)
	})

	Client.OnSelfPartMessage(func(m twitch.UserPartMessage) {
		log.Println("parted", m.Channel)
		setConfirmed(m.Channel, false)
	})

	Client.OnPrivateMessage(func(m twitch.PrivateMessage) {
		if m.User.ID == config.Auth.Twitch.GQL.UserID {
			return
		}

		if IsBlocked(m.User.ID) {
			return
		}

		for _, cmd := range Fun.Cmds {
			go func(cmd Cmd) {
				channels := config.Meta.Functions[cmd.Name].Channels
				if len(channels) > 0 && !slices.Contains(channels, m.RoomID) {
					return
				}

				err := cmd.Handler(m)
				if err != nil {
					log.Printf("[cmd error] %v: %v => %v\n", m.User.Name, m.Message, err)
					api_twitch.Say(m.RoomID, "🚫 "+err.Error(), m.ID)
				}
			}(cmd)
		}
	})

	for _, userID := range config.Meta.Channels {
		if !joinChannel(userID) {
			go func(userID string) {
				for delay := 5 * time.Second; !joinChannel(userID); delay = min(delay*2, 5*time.Minute) {
					time.Sleep(delay)
				}
			}(userID)
		}
	}

	go retryUnconfirmedJoins()

	go func() {
		err := Client.Connect()
		if err != nil {
			log.Panicln("failed connecting to Twitch IRC", err)
		}
	}()
}

// joinChannel reports false only when the lookup failed and is worth retrying.
func joinChannel(userID string) bool {
	u, err := api_twitch.GetUser("", userID)
	if err != nil {
		log.Println("failed getting user", userID, err)
		return false
	}
	if u.Login == "" {
		log.Println("user not found", userID)
		return true
	}
	Join(u.Login)
	return true
}
