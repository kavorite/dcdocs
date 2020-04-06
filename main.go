package main

import (
	"flag"
	"fmt"
	"github.com/kavorite/discord-snowflake"
	"github.com/pkg/browser"
	"io"
	"math"
	"os"
	"strings"
	"time"

	dgo "github.com/bwmarrin/discordgo"
	_ "github.com/kavorite/discord-spool"
)

type Op string

const (
	OpLogin          Op = "login"
	OpHydrateChannel Op = "channel.hydrate"
	OpResolveChannel Op = "channel.resolve"
	OpResolveUser    Op = "user.resolve"
	OpParseTimespan  Op = "timespan.parse"
	OpOpenURI        Op = "uri.open"
)

type Error struct {
	Op
	Cause error
}

func (err Error) Error() string {
	return fmt.Sprintf("%s: %s", err.Op, err.Cause)
}

func (err Error) FCk() {
	if err.Cause != nil {
		panic(fmt.Errorf("fatal: %s", err))
	}
}

var (
	token, tag      string
	lookbehind      string
	targetSnowflake uint64
	targetDM        string
	printOnly       bool
)

func timespan(src string) (span time.Duration, err error) {
	var (
		s float64
		u rune
	)
	istrm := strings.NewReader(src)
	for {
		var n int
		n, err = fmt.Fscanf(istrm, "%f%c", &s, &u)
		unit := time.Second
		if n > 1 {
			switch u {
			case 'd':
				unit *= 24
				fallthrough
			case 'h':
				unit *= 60
				fallthrough
			case 'm':
				unit *= 60
				fallthrough
			case 's':
				unit *= 1
			default:
				err = fmt.Errorf("unit '%c' not recognized", u)
				return
			}
		}
		span += time.Duration(math.Round(s * float64(unit)))
		if err != nil {
			if err == io.EOF {
				err = nil
			}
			return
		}
	}
}

func main() {
	flag.StringVar(&token, "T", "", "Discord authentication token")
	flag.BoolVar(&printOnly, "p", false, "[p]rint-only")
	flag.StringVar(&targetDM, "g", "", "Target DM name")
	flag.Uint64Var(&targetSnowflake, "t", 0, "Target user or channel snowflake")
	flag.StringVar(&lookbehind, "d", "1d",
		"how much history to grab in units of "+
			"[s]econds, [m]inutes, [h]ours, and [d]ays (e.g.: 1h30m)")
	flag.Parse()
	targetDM = strings.Trim(targetDM, `"'`)
	if token == "" {
		token = os.Getenv("DCDOC_TOKEN")
	}
	token = strings.Trim(token, `"'`)
	if token == "" {
		Error{OpLogin, fmt.Errorf("Missing authentication token; please provide one of DCDOC_TOKEN or -T")}.FCk()
	}
	maxAge, err := timespan(lookbehind)
	Error{OpParseTimespan, err}.FCk()
	client, err := dgo.New(token)
	Error{OpLogin, err}.FCk()
	if targetSnowflake == 0 && targetDM == "" {
		Error{
			OpResolveChannel,
			fmt.Errorf("No target specified; please provide one of -g, -u, or -t"),
		}.FCk()
	}
	var (
		target *dgo.Channel
		dms    []*dgo.Channel
	)
	if targetDM != "" {
		dms, err = client.UserChannels()
		Error{OpResolveChannel, err}.FCk()
		if len(dms) == 0 {
			return
		}
		target = dms[0]
		targetDistance := lev(targetDM, target.Recipients[0].String())
		for _, dm := range dms {
			if dm.Type != dgo.ChannelTypeDM || len(dm.Recipients) != 1 {
				continue
			}
			distance := lev(targetDM, dm.Recipients[0].String())
			if distance < targetDistance {
				target = dm
				targetDistance = distance
			}
		}
	} else if targetSnowflake != 0 {
		// try retrieving the snowflake as a channel by ID
		target, err = client.Channel(fmt.Sprintf("%d", targetSnowflake))
		// if 404, try resolving this snowflake against a one-to-one DM channel
		// instead
		msg := err.Error()
		pfx := "HTTP 404 Not Found"
		if pfx == msg[:len(pfx)] {
			if len(dms) == 0 {
				dms, err = client.UserChannels()
				Error{OpResolveChannel, err}.FCk()
			}
			for _, dm := range dms {
				if dm.Type != dgo.ChannelTypeDM || len(dm.Recipients) != 1 {
					continue
				} else if dm.Recipients[0].ID == fmt.Sprintf("%d", targetSnowflake) {
					target = dm
				}
			}
			if target == nil {
				Error{OpResolveChannel, fmt.Errorf("No private channel found matching recipient")}.FCk()
			}
		} else {
			Error{OpResolveChannel, err}.FCk()
		}
	}
	after := fmt.Sprintf("%d", snowflake.T(0).Stamp(time.Now().Add(-maxAge)))
	before := "00000000000000000000"
	forEach := func(msg *dgo.Message) {
		for _, doc := range msg.Attachments {
			if printOnly {
				fmt.Println(doc.URL)
			} else {
				err := browser.OpenURL(doc.URL)
				Error{OpOpenURI, err}.FCk()
			}
		}
	}
	msgs, err := client.ChannelMessages(target.ID, 1, "", "", "")
	Error{OpHydrateChannel, err}.FCk()
	if len(msgs) > 0 {
		forEach(msgs[0])
	}
	for {
		msgs, err := client.ChannelMessages(target.ID, 100, before, after, "")
		Error{OpHydrateChannel, err}.FCk()
		if len(msgs) > 0 {
			after = msgs[0].ID
			for i := len(msgs) - 1; i > 0; i-- {
				forEach(msgs[i])
			}
		} else {
			return
		}
	}
}
