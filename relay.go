package main

import (
	"flag"
	"log"
	"os/user"
	"strings"

	"github.com/velour/relay/irc"
	"github.com/velour/relay/slack"
)

var (
	ircServer   = flag.String("ircserver", "irc.freenode.net:7000", "The IRC host and port")
	ircSSL      = flag.Bool("ircssl", true, "Whether to use SSL to connect to the IRC server")
	ircPassword = flag.String("ircpassword", "", "The password for the IRC server")
	ircNick     = flag.String("ircnick", nick(), "The IRC nick name")
	ircFullName = flag.String("ircfullname", fullname(), "The IRC full name")
	ircChannel  = flag.String("ircchannel", "", "The IRNC channel to relay")
)

var (
	slackToken   = flag.String("slacktoken", "", "The slack token")
	slackNick    = flag.String("slacknick", nick(), "The username to relay into IRC")
	slackChannel = flag.String("slackchannel", "", "The channel (with # prefix) or private channel (no # prefix)")
)

func nick() string {
	un, err := user.Current()
	if err != nil {
		return ""
	}
	return un.Username
}

func fullname() string {
	un, err := user.Current()
	if err != nil || un.Name == "" {
		return un.Username
	}
	return un.Name
}

func main() {
	flag.Parse()

	fromIRC := make(chan message)
	ircClient := startIRC(fromIRC)
	defer ircClient.Close()
	log.Println("irc connected")

	fromSlack := make(chan message)
	slackClient, channelID := startSlack(fromSlack)
	defer slackClient.Close()
	log.Println("slack connected")

	for {
		select {
		case msg := <-fromSlack:
			if err := ircClient.Send(irc.PRIVMSG, *ircChannel, msg.text); err != nil {
				log.Println("irc failed to send PRIVMSG:", err)
			}
		case msg := <-fromIRC:
			server := strings.SplitN(*ircServer, ":", 2)[0]
			var who string
			if msg.who == "" {
				who = server
			} else {
				who = msg.who + "@" + server
			}
			if err := slackClient.PostMessage(who, channelID, msg.text); err != nil {
				log.Println("slack failed to post message:", err)
			}
		}
	}
}

type message struct {
	who     string
	channel string
	text    string
}

func startSlack(ch chan<- message) (c *slack.Client, channelID string) {
	c, err := slack.NewClient(*slackToken)
	if err != nil {
		log.Fatalln("slack failed to connect:", err)
	}

	users, err := c.UsersList()
	if err != nil {
		log.Fatalln("slack failed to get users list:", err)
	}
	var userID string
	for _, u := range users {
		if u.Name == *slackNick {
			userID = u.ID
			break
		}
	}
	if userID == "" {
		log.Fatalln("slack no user:", *slackNick)
	}

	var channels []slack.Channel
	if strings.HasPrefix(*slackChannel, "#") {
		channels, err = c.ChannelsList()
	} else {
		channels, err = c.GroupsList()
	}
	for _, c := range channels {
		if c.Name == strings.TrimPrefix(*slackChannel, "#") {
			channelID = c.ID
			break
		}
	}
	if channelID == "" {
		log.Fatalln("slack no channel:", *slackChannel)
	}

	go func() {
		defer close(ch)
		for {
			event, err := c.Next()
			if err != nil {
				log.Fatalln("failed to read slack event:", err)
				return
			}
			switch t, _ := event["type"].(string); t {
			case "message":
				if _, ok := event["subtype"]; ok {
					break
				}
				if _, ok := event["reply_to"]; ok {
					break
				}
				channel, _ := event["channel"].(string)
				user, _ := event["user"].(string)
				text, _ := event["text"].(string)
				if channel != channelID || user != userID {
					continue
				}
				log.Printf("slack sending message\n%#v\n\n", event)
				ch <- message{who: *slackNick, channel: *slackChannel, text: text}
			case "presence_change",
				"reconnect_url",
				"user_typing":
				// Silence noisy events.
			default:
				log.Printf("slack event:\n%#v\n\n", event)
			}
		}
	}()

	return c, channelID
}

func startIRC(ch chan<- message) *irc.Client {
	var err error
	var c *irc.Client
	if *ircSSL {
		c, err = irc.DialSSL(*ircServer, *ircNick, *ircFullName, *ircPassword, false)
	} else {
		c, err = irc.Dial(*ircServer, *ircNick, *ircFullName, *ircPassword)
	}
	if err != nil {
		log.Fatalln("irc failed to dial:", err)
	}

	if err := c.Send(irc.JOIN, *ircChannel); err != nil {
		log.Fatalln("irc failed to send JOIN:", err)
	}

	go func() {
		defer close(ch)
		for {
			msg, err := c.Next()
			if err != nil {
				log.Fatalln("IRC read error:", err)
				return
			}
			switch msg.Command {
			case irc.JOIN:
				if len(msg.Arguments) < 1 {
					break
				}
				who := msg.Origin
				channel := msg.Arguments[0]
				if channel != *ircChannel {
					break
				}
				ch <- message{channel: channel, text: who + " joined"}

			case irc.PRIVMSG:
				if len(msg.Arguments) < 2 {
					break
				}
				who := msg.Origin
				channel := msg.Arguments[0]
				text := msg.Arguments[1]
				if channel != *ircChannel {
					break
				}
				if who == *ircNick {
					break
				}
				ch <- message{who: who, channel: channel, text: text}
			default:
				log.Printf("irc message:\n%#v\n\n", msg)
			}
		}
	}()

	return c
}
