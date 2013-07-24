// Copyright 2009 Thomas Jager <mail@jager.no>  All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package irc

import (
	"github.com/daviddengcn/go-villa"
	"strings"
)

const (
	NICK    = "NICK"
	USER    = "USER"
	PASS    = "PASS"
	JOIN    = "JOIN"
	PART    = "PART"
	QUIT    = "QUIT"
	NOTICE  = "NOTICE"
	PRIVMSG = "PRIVMSG"
	PING    = "PING"
	PONG    = "PONG"
	TIME    = "TIME"
	MODE    = "MODE"
	ERROR   = "ERROR"
	VERSION = "VERSION"

	CLIENTINFO      = "CLIENTINFO"
	USERINFO        = "USERINFO"
	CTCP            = "CTCP"
	CTCP_VERSION    = "CTCP_VERSION"
	CTCP_TIME       = "CTCP_TIME"
	CTCP_PING       = "CTCP_PING"
	CTCP_USERINFO   = "CTCP_USERINFO"
	CTCP_CLIENTINFO = "CTCP_CLIENTINFO"

	RPL_WELCOME       = "001"
	RPL_YOURHOST      = "002"
	RPL_CREATED       = "003"
	RPL_MYINFO        = "004"
	RPL_ISUPPORT      = "005"
	RPL_STATSCONN     = "250"
	RPL_LUSERCLIENT   = "251"
	RPL_LUSEROP       = "252"
	RPL_LUSERUNKNOWN  = "253"
	RPL_LUSERCHANNELS = "254"
	RPL_LUSERME       = "255"
	RPL_LOCALUSERS    = "265"
	RPL_GLOBALUSERS   = "266"
	RPL_TOPIC         = "332"
	RPL_NAMREPLY      = "353"
	RPL_ENDOFNAMES    = "366"
	RPL_MOTD          = "372"
	RPL_MOTDSTART     = "375"
	RPL_ENDOFMOTD     = "376"

	ERR_NICKNAMEINUSE = "433"
	ERR_BANNICKCHANGE = "437"
)

type Event struct {
	Raw string // raw message line received. contains all of the following information

	Source string // <nick>!<usr>@<host>
	Nick   string // <nick>
	User   string // <usr>
	Host   string // <host>

	Code      string   // PING/PONG ...
	Arguments []string // arguments to Code

	Message string // message after " :"
}

func (e *Event) setByLine(line string) {
	e.Raw = line
	if line[0] == ':' {
		if p := strings.Index(line, " "); p >= 0 {
			e.Source = line[1:p]
			line = line[p+1:]

			if i, j := strings.Index(e.Source, "!"), strings.Index(e.Source, "@"); i >= 0 && j >= 0 && i < j {
				e.Nick = e.Source[:i]
				e.User = e.Source[i+1 : j]
				e.Host = e.Source[j+1:]
			}
		}
	}

	if p := strings.Index(line, " :"); p >= 0 {
		e.Message = line[p+2:]
		line = line[:p]
	}

	args := strings.Split(line, " ")
	e.Code = strings.ToUpper(args[0])

	e.Arguments = args[1:]
}

func composeMessage(code string, message string, params ...string) string {
	var buf villa.ByteSlice
	buf.WriteString(code)
	for _, p := range params {
		buf.WriteRune(' ')
		buf.WriteString(p)
	}
	if len(message) > 0 {
		buf.WriteString(" :")
		buf.WriteString(message)
	}
	return string(buf)
}
