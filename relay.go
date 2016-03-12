package main

import (
	"bufio"
	"flag"
	"log"
	"os"
	"os/user"

	"github.com/velour/relay/irc"
)

var (
	ircserver   = flag.String("ircserver", "irc.freenode.net:7000", "The IRC host and port")
	ircssl      = flag.Bool("ircssl", true, "Whether to use SSL to connect to the IRC server")
	ircpass     = flag.String("ircpass", "", "The password for the IRC server")
	ircnick     = flag.String("ircnick", nick(), "The IRC nick name")
	ircfullname = flag.String("ircfullname", fullname(), "The IRC full name")
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

	var ircClient *irc.Client
	var err error
	if *ircssl {
		ircClient, err = irc.DialSSL(*ircserver, *ircnick, *ircfullname, *ircpass, false)
	} else {
		ircClient, err = irc.Dial(*ircserver, *ircnick, *ircfullname, *ircpass)
	}
	if err != nil {
		log.Fatalln("failed to dial IRC: ", err)
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
