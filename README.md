# relay
Relay forwards all messages from an IRC channel to a slack channel,
and all messages from a single user in the slack channel back to the IRC channel.

```
$ relay -help
Usage of relay:
  -ircchannel string
        The IRNC channel to relay
  -ircfullname string
        The IRC full name
  -ircnick string
        The IRC nick name
  -ircpassword string
        The password for the IRC server
  -ircserver string
        The IRC host and port (default "irc.freenode.net:7000")
  -ircssl
        Whether to use SSL to connect to the IRC server (default true)
  -slackchannel string
        The channel (with # prefix) or private channel (no # prefix)
  -slacknick string
        The username to relay into IRC
  -slacktoken string
        The slack token
```
