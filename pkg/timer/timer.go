package timer

import (
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// WaitTimer is used to call a given method once the wait time has been reached.
type WaitTimer struct {
	s            *discordgo.Session
	i            *discordgo.InteractionCreate
	timerChannel chan int
	methodToCall func(s *discordgo.Session, i *discordgo.InteractionCreate)
	expiration   time.Time
}

// NewWaitTimer creates a waitTimer with the given configuration information.
func NewWaitTimer(s *discordgo.Session, i *discordgo.InteractionCreate, waitTime time.Duration, msgFunc func(*discordgo.Session, *discordgo.InteractionCreate, string) error, methodToCall func(*discordgo.Session, *discordgo.InteractionCreate)) *WaitTimer {
	timerChannel := make(chan int)
	expiration := time.Now().Add(waitTime)
	t := WaitTimer{
		s:            s,
		i:            i,
		timerChannel: timerChannel,
		methodToCall: methodToCall,
		expiration:   expiration,
	}
	go t.Start(msgFunc)
	return &t
}

// Start starts the wait timer. Once it expires, `methodToCall` is called. The timer
// can be cancelled by calling `canel()`.
func (t *WaitTimer) Start(msgFunc func(s *discordgo.Session, i *discordgo.InteractionCreate, action string) error) {
	// Update the message every five seconds with the new expiration time until the
	// time has expired.
	for !time.Now().After(t.expiration) {
		maximumWait := time.Until(t.expiration)
		timeToWait := min(maximumWait, 5*time.Second)
		if timeToWait < 0 {
			break
		}
		wait := time.NewTimer(timeToWait)
		select {
		case <-wait.C:
			err := msgFunc(t.s, t.i, "update")
			if err != nil {
				log.Error("Unable to update the time for the heist message, error:", err)
			}
		case <-t.timerChannel:
			if wait.Stop() {
				<-wait.C
			}
			return
		}
	}
	t.methodToCall(t.s, t.i)
}

// Cancel disables the wait timer.
func (t *WaitTimer) Cancel() {
	log.Debug("--> timer.cancel")
	defer log.Debug("<-- timer.cancel")
	t.timerChannel <- 1
}
