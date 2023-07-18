package heist

import (
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// waitTimer is used to call a given method once the wait time has been reached.
type waitTimer struct {
	s            *discordgo.Session
	i            *discordgo.InteractionCreate
	timerChannel chan int
	methodToCall func(s *discordgo.Session, i *discordgo.InteractionCreate)
	wait         *time.Timer
}

// newWaitTimer creates a waitTimer with the given configuration information.
func newWaitTimer(s *discordgo.Session, i *discordgo.InteractionCreate, waitTime time.Duration, methodToCall func(*discordgo.Session, *discordgo.InteractionCreate)) *waitTimer {
	timerChannel := make(chan int)
	t := waitTimer{
		s:            s,
		i:            i,
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
		t.methodToCall(t.s, t.i)
	case <-t.timerChannel:
		if t.wait.Stop() {
			<-t.wait.C
		}
	}
}

// cancel disables the wait timer.
func (t *waitTimer) cancel() {
	log.Debug("--> timer.cancel")
	defer log.Debug("<-- timer.cancel")
	t.timerChannel <- 1
}
