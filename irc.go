// Copyright 2013 David Deng All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package irc provides a IRC client, implementing IRC protocol, for
	communicating with an IRC server.
*/
package irc

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	IRC_VERSION = "go-irc"
)

/*
 */
type Client struct {
	Password  string
	TLSConfig *tls.Config

	socket  net.Conn
	cRead   chan string
	cWrite  chan string
	cError  chan error
	cExit   chan bool
	endPing chan bool

	nick        string //The nickname we want.
	currentNick string //The nickname we currently have.
	username    string
	registered  bool

	DefaultHandler func(*Event)
	handlers       map[string]func(*Event)

	lastMessage time.Time
}

func (c *Client) readLoop() {
	br := bufio.NewReaderSize(c.socket, 512)

	for {
		line, err := br.ReadString('\n')
		if err != nil {
			c.cError <- err
			break
		}

		c.lastMessage = time.Now()
		line = line[:len(line)-2] // Remove \r\n

		e := &Event{}
		e.setByLine(line)

		c.handle(e)
	}

	c.cExit <- true
}

func (c *Client) writeLoop() {
	for msg := range c.cWrite {
		if c.socket == nil {
			break
		}

		if _, err := c.socket.Write([]byte(msg + "\r\n")); err != nil {
			c.cError <- err
			break
		}
	}
	c.cExit <- true
}

// Ping sends a Ping command to IRC server.
func (c *Client) Ping() {
	c.Command(PING, "", strconv.FormatInt(time.Now().UnixNano(), 10))
}

//Pings the server if we have not recived any messages for 5 minutes
func (c *Client) pingLoop() {
	ticker := time.NewTicker(1 * time.Minute)   //Tick every minute.
	ticker2 := time.NewTicker(15 * time.Minute) //Tick every 15 minutes.
	for {
		select {
		case <-ticker.C:
			//Ping if we haven't received anything from the server within 4 minutes
			if time.Since(c.lastMessage) >= (4 * time.Minute) {
				c.Ping()
			}
		case <-ticker2.C:
			//Ping every 15 minutes.
			c.Ping()
			//Try to recapture nickname if it's not as configured.
			if c.nick != c.currentNick {
				c.currentNick = c.nick
				c.Command(NICK, "", c.nick)
			}
		case <-c.endPing:
			ticker.Stop()
			ticker2.Stop()
			c.cExit <- true
			return
		}
	}
}

// Disconnect sends all buffered messages (if possible), stops all goroutines
// and then closes the socket.
func (c *Client) Disconnect() {
	c.endPing <- true
	close(c.cWrite)
	close(c.cRead)

	<-c.cExit
	<-c.cExit
	<-c.cExit

	c.socket.Close()
	c.socket = nil

	close(c.cError)
}

func (c *Client) Quit() {
	c.Command(QUIT, "")
	c.Disconnect()
}

func (c *Client) Serve() error {
	return <-c.cError
}

func (c *Client) Join(channels ...string) {
	c.Command(JOIN, "", strings.Join(channels, ","))
}

func (c *Client) Part(channels ...string) {
	c.Command(PART, "", strings.Join(channels, ","))
}

func (c *Client) Notice(target, message string) {
	c.Command(NOTICE, message, target)
}

func (c *Client) Noticef(target, format string, a ...interface{}) {
	c.Notice(target, fmt.Sprintf(format, a...))
}

func (c *Client) Privmsg(target, message string) {
	c.Command(PRIVMSG, message, target)
}

func (c *Client) Privmsgf(target, format string, a ...interface{}) {
	c.Privmsg(target, fmt.Sprintf(format, a...))
}

func (c *Client) Raw(message string) {
	c.cWrite <- message
}

func (c *Client) Rawf(format string, a ...interface{}) {
	c.cWrite <- fmt.Sprintf(format, a...)
}

func (c *Client) Command(code string, message string, params ...string) {
	c.cWrite <- composeMessage(code, message, params...)
}

// SetNick sets the nickname expected to be used in the channel. According to
// IRC protocol, the nickname specified may be not accepted by the server. Call
// Client.Nick() to get the current used nickname.
func (c *Client) SetNick(nick string) {
	c.nick = nick
	c.Command(NICK, "", nick)
}

// Nick returns the current used nickname.
func (c *Client) Nick() string {
	return c.currentNick
}

// Dial connects to the server and starts the Client
func (c *Client) Dial(server string) (err error) {
	var socket net.Conn
	if c.TLSConfig == nil {
		if socket, err = net.Dial("tcp", server); err != nil {
			return err
		}
	} else {
		if socket, err = tls.Dial("tcp", server, c.TLSConfig); err != nil {
			return err
		}
	}
	c.Start(socket)
	return nil
}

// Starts bind a connected socket(net.Conn) to the Client
func (c *Client) Start(socket net.Conn) {
	c.socket = socket

	c.cRead = make(chan string, 10)
	c.cWrite = make(chan string, 10)
	c.cError = make(chan error, 2)

	go c.readLoop()
	go c.writeLoop()
	go c.pingLoop()

	if len(c.Password) > 0 {
		c.Command(PASS, "", c.Password)
	}
	c.Command(NICK, "", c.nick)
	c.Command(USER, c.username, c.username, "0.0.0.0 0.0.0.0")
}

func (c *Client) SetHandler(code string, handler func(*Event)) {
	code = strings.ToUpper(code)
	c.handlers[code] = handler
}

func (c *Client) handle(event *Event) {
	if event.Code == PRIVMSG && len(event.Message) > 0 && event.Message[0] == '\x01' {
		event.Code = CTCP // Unknown CTCP

		if i := strings.LastIndex(event.Message, "\x01"); i > -1 {
			event.Message = event.Message[1:i]
		}

		if event.Message == VERSION {
			event.Code = CTCP_VERSION

		} else if event.Message == TIME {
			event.Code = CTCP_TIME

		} else if len(event.Message) >= 4 && event.Message[:4] == PING {
			event.Code = CTCP_PING

		} else if event.Message == USERINFO {
			event.Code = CTCP_USERINFO

		} else if event.Message == CLIENTINFO {
			event.Code = CTCP_CLIENTINFO
		}
	}

	if handler, ok := c.handlers[event.Code]; ok {
		go handler(event)
	} else if c.DefaultHandler != nil {
		c.DefaultHandler(event)
	}
}

func (c *Client) setupCallbacks() {
	//Handle ping events
	c.SetHandler(PING, func(e *Event) {
		c.Command(PONG, e.Message)
	})

	//Version handler
	c.SetHandler(CTCP_VERSION, func(e *Event) {
		c.Noticef(e.Nick, "\x01\x01VERSION %s\x01", IRC_VERSION)
	})

	c.SetHandler(CTCP_USERINFO, func(e *Event) {
		c.Noticef(e.Nick, "\x01\x01USERINFO %s\x01", c.username)
	})

	c.SetHandler(CTCP_CLIENTINFO, func(e *Event) {
		c.Notice(e.Nick, "\x01CLIENTINFO PING VERSION TIME USERINFO CLIENTINFO\x01")
	})

	c.SetHandler(CTCP_TIME, func(e *Event) {
		c.Noticef(e.Nick, "\x01TIME %s\x01", time.Now().String())
	})

	c.SetHandler(CTCP_PING, func(e *Event) {
		c.Noticef(e.Nick, "\x01%s\x01", e.Message)
	})

	c.SetHandler(ERR_BANNICKCHANGE, func(e *Event) {
		c.currentNick += "_"
		c.Command(NICK, "", c.currentNick)
	})

	c.SetHandler(ERR_NICKNAMEINUSE, func(e *Event) {
		if len(c.currentNick) > 8 {
			c.currentNick = "_" + c.currentNick
		} else {
			c.currentNick = c.currentNick + "_"
		}
		c.Command(NICK, "", c.currentNick)
	})

	c.SetHandler(NICK, func(e *Event) {
		if e.Nick == c.nick {
			c.currentNick = e.Message
		}
	})

	c.SetHandler(RPL_WELCOME, func(e *Event) {
		c.currentNick = e.Arguments[0]
	})
}

// NewClient creates a new Client instance, disconnected.
func NewClient(nick, username string) *Client {
	c := &Client{
		nick:     nick,
		username: username,
		cExit:    make(chan bool),
		endPing:  make(chan bool),
		handlers: make(map[string]func(*Event)),
	}
	c.setupCallbacks()
	return c
}
