package irc

import (
	//	"github.com/thoj/go-ircevent"
	"testing"
)

func TestConnection(t *testing.T) {
	clinet := NewClient("go-eventirc", "go-eventirc")
	if err := clinet.Dial("irc.freenode.net:6667"); err != nil {
		t.Fatal("Can't connect to freenode.")
	}
	clinet.SetHandler(RPL_WELCOME, func(e *Event) { clinet.Join("#go-eventirc") })

	clinet.SetHandler(RPL_ENDOFNAMES, func(e *Event) {
		clinet.Privmsg("#go-eventirc", "Test Message\n")
		clinet.SetNick("go-eventnewnick")
	})
	clinet.SetHandler(NICK, func(e *Event) {
		clinet.Quit()
		if clinet.currentNick == "go-eventnewnick" {
			t.Fatal("Nick change did not work!")
		}
	})
	clinet.Serve()
}

/*
func TestConnectionSSL(t *testing.T) {
	irccon := IRC("go-eventirc", "go-eventirc")
	irccon.VerboseCallbackHandler = true
	irccon.UseTLS = true
	err := irccon.Connect("irc.freenode.net:7000")
	if err != nil {
		t.Fatal("Can't connect to freenode.")
	}
	irccon.AddCallback("001", func(e *Event) { irccon.Join("#go-eventirc") })

	irccon.AddCallback("366", func(e *Event) {
		irccon.Privmsg("#go-eventirc", "Test Message\n")
		irccon.Quit()
	})

	irccon.Loop()
}
*/
