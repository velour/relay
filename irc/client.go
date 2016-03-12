package irc

import (
	"bufio"
	"crypto/tls"
	"errors"
	"net"
	"time"
)

// A Client is a client's connection to an IRC server.
type Client struct {
	conn net.Conn
	in   *bufio.Reader
}

// Dial connects to a remote IRC server.
func Dial(server, nick, fullname, pass string) (*Client, error) {
	c, err := net.Dial("tcp", server)
	if err != nil {
		return nil, err
	}
	return dial(c, nick, fullname, pass)
}

// DialSSL connects to a remote IRC server using SSL.
func DialSSL(server, nick, fullname, pass string, trust bool) (*Client, error) {
	c, err := tls.Dial("tcp", server, &tls.Config{InsecureSkipVerify: trust})
	if err != nil {
		return nil, err
	}
	return dial(c, nick, fullname, pass)
}

func dial(conn net.Conn, nick, fullname, pass string) (*Client, error) {
	c := &Client{conn: conn, in: bufio.NewReader(conn)}
	if err := register(c, nick, fullname, pass); err != nil {
		return nil, err
	}
	return c, nil
}

func register(c *Client, nick, fullname, pass string) error {
	if pass != "" {
		if err := c.Write(PASS, pass); err != nil {
			return err
		}
	}
	if err := c.Write(NICK, nick); err != nil {
		return err
	}
	if err := c.Write(USER, nick, "0", "*", fullname); err != nil {
		return err
	}
	for {
		msg, err := c.Read()
		if err != nil {
			return err
		}
		switch msg.Command {
		case ERR_NONICKNAMEGIVEN, ERR_ERRONEUSNICKNAME,
			ERR_NICKNAMEINUSE, ERR_NICKCOLLISION,
			ERR_UNAVAILRESOURCE, ERR_RESTRICTED,
			ERR_NEEDMOREPARAMS, ERR_ALREADYREGISTRED:
			if len(msg.Arguments) > 0 {
				return errors.New(msg.Arguments[len(msg.Arguments)-1])
			}
			return errors.New(CommandNames[msg.Command])

		case RPL_WELCOME:
			return nil

		default:
			/* ignore */
		}
	}
}

// Close closes the connection.
func (c *Client) Close() error {
	c.Write(QUIT)
	return c.conn.Close()
}

// WriteMessage writes a message to the server.
func (c *Client) WriteMessage(msg Message) error {
	deadline := time.Now().Add(time.Minute)
	if err := c.conn.SetWriteDeadline(deadline); err != nil {
		return err
	}
	bs := msg.Bytes()
	if len(bs) > MaxBytes {
		return TooLongError{Message: bs[:MaxBytes], NTrunc: len(bs) - MaxBytes}
	}
	_, err := c.conn.Write(bs)
	return err
}

// Write writes a message to the server with the given command and arguments.
func (c *Client) Write(cmd string, args ...string) error {
	return c.WriteMessage(Message{Command: cmd, Arguments: args})
}

// Read reads the next message from the server.
// Read never returns a PING command;
// the client responds to PINGs automatically.
func (c *Client) Read() (Message, error) {
	for {
		switch msg, err := read(c.in); {
		case err != nil:
			return Message{}, err
		case msg.Command == PING:
			if err := c.Write(PONG, msg.Arguments...); err != nil {
				return Message{}, err
			}
		default:
			return msg, nil
		}
	}
}
