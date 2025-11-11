package fun

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/0supa/supa8/config"
	. "github.com/0supa/supa8/fun"
	. "github.com/0supa/supa8/fun/api/twitch"
	"github.com/gempir/go-twitch-irc/v4"
)

func runCommands(cmds [][]string) (res []string, err error) {
	for _, c := range cmds {
		if len(c) == 0 {
			continue
		}
		cmd := exec.Command(c[0], c[1:]...)
		b, err := cmd.CombinedOutput()
		if err != nil {
			return res, fmt.Errorf("command %v failed: %w; output: %s", c[0], err, string(b))
		}
		res = append(res, string(b))
	}
	return
}

func init() {
	Fun.Register(&Cmd{
		Name: "ping",
		Handler: func(m twitch.PrivateMessage) (err error) {
			if m.Message != "`ping" {
				return
			}

			msPing := time.Since(m.Time).Milliseconds()

			var mem runtime.MemStats
			runtime.ReadMemStats(&mem)

			res, err := runCommands([][]string{
				{"git", "rev-parse", "--short", "HEAD"},
				{"yt-dlp", "--version"},
				{"ffmpeg", "-version"},
				{"zbarimg", "--version"},
			})
			if err != nil {
				return
			}

			var ffmpegVer string
			fmt.Sscanf(res[2], "ffmpeg version %s", &ffmpegVer)

			_, err = Say(m.RoomID, fmt.Sprintf("pong!! %vms, %vMiB, up:%s, channels:%v, blocked:%v, commit:%s, %s, yt-dlp%s, ffmpeg%s, zbar%s",
				msPing,
				mem.Alloc/1024/1024,
				time.Since(InitTime).Truncate(time.Second),
				len(config.Meta.Channels),
				len(Fun.BlockedUserIDs),
				strings.TrimSpace(res[0]),
				runtime.Version(),
				strings.TrimSpace(res[1]),
				ffmpegVer,
				strings.TrimSpace(res[3]),
			), m.ID)
			return
		},
	})
}
