package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"
	"os/user"

	"github.com/velour/relay/irc"
	"github.com/velour/relay/slack"
)

var (
	ircServer   = flag.String("ircserver", "irc.freenode.net:7000", "The IRC host and port")
	ircSSL      = flag.Bool("ircssl", true, "Whether to use SSL to connect to the IRC server")
	ircPassword = flag.String("ircpassword", "", "The password for the IRC server")
	ircNick     = flag.String("ircnick", nick(), "The IRC nick name")
	ircFullName = flag.String("ircfullname", fullname(), "The IRC full name")
)

var (
	slackToken = flag.String("slacktoken", "", "The slack token")
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

	doSlack()
}

func doSlack() {
	slackClient, err := slack.NewClient(*slackToken)
	if err != nil {
		log.Fatalln("failed to connect to slack:", err)
	}
	fmt.Println(slackClient.UsersList())
	fmt.Println(slackClient.ChannelsList())
	fmt.Println(slackClient.GroupsList())
	for {
		event, err := slackClient.Next()
		if err != nil {
			log.Fatalln("failed to read slack event:", err)
		}
		log.Printf("%#v\n", event)
	}
}

func doIRC() {
	var ircClient *irc.Client
	var err error
	if *ircSSL {
		ircClient, err = irc.DialSSL(*ircServer, *ircNick, *ircFullName, *ircPassword, false)
	} else {
		ircClient, err = irc.Dial(*ircServer, *ircNick, *ircFullName, *ircPassword)
	}
	if err != nil {
		log.Fatalln("failed to dial IRC:", err)
	}
	defer ircClient.Close()

	go func() {
		for {
			msg, err := ircClient.Read()
			if err != nil {
				return
			}
			log.Println(string(msg.Bytes()))
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		msg, err := irc.Parse(scanner.Bytes())
		if err != nil {
			log.Println(err)
			continue
		}
		if err := ircClient.WriteMessage(msg); err != nil {
			log.Println(err)
			break
		}
	}
	if err := scanner.Err(); err != nil {
		log.Println("error reading standard input: ", err)
	}
}
