package heist

import (
	"time"

	"github.com/bwmarrin/discordgo"
)

// waitTimer is used to call a given method once the wait time has been reached.
type waitTimer struct {
	s            *discordgo.Session
	server       *Server
	channelID    string
	messageID    string
	timerChannel chan int
	methodToCall func(s *discordgo.Session, server *Server, channelID string, messageID string)
	wait         *time.Timer
}

// newWaitTimer creates a waitTimer with the given configuration information.
func newWaitTimer(s *discordgo.Session, server *Server, channelID string, messageID string, waitTime time.Duration, methodToCall func(*discordgo.Session, *Server, string, string)) *waitTimer {
	timerChannel := make(chan int)
	t := waitTimer{
		s:            s,
		server:       server,
		channelID:    channelID,
		messageID:    messageID,
		timerChannel: timerChannel,
		methodToCall: methodToCall,
		wait:         time.NewTimer(waitTime),
	}
	go t.start()
	return &t
}

// start starts the wait timer. Once it expires, `methodToCall` is called. The timer
// can be cancelled by calling `canel()`.
func (t *waitTimer) start() {
	select {
	case <-t.wait.C:
		t.methodToCall(t.s, t.server, t.channelID, t.messageID)
	case <-t.timerChannel:
		if t.wait.Stop() {
			<-t.wait.C
		}
	}
}

// cancel disables the wait timer.
func (t *waitTimer) cancel() {
	t.timerChannel <- 1
}
