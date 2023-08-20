package heist

import (
	"time"

	"github.com/rbrabson/heist/pkg/math"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// waitTimer is used to call a given method once the wait time has been reached.
type waitTimer struct {
	s            *discordgo.Session
	i            *discordgo.InteractionCreate
	timerChannel chan int
	methodToCall func(s *discordgo.Session, i *discordgo.InteractionCreate)
	expiration   time.Time
}

// newWaitTimer creates a waitTimer with the given configuration information.
func newWaitTimer(s *discordgo.Session, i *discordgo.InteractionCreate, waitTime time.Duration, methodToCall func(*discordgo.Session, *discordgo.InteractionCreate)) *waitTimer {
	timerChannel := make(chan int)
	expiration := time.Now().Add(waitTime)
	t := waitTimer{
		s:            s,
		i:            i,
		timerChannel: timerChannel,
		methodToCall: methodToCall,
		expiration:   expiration,
	}
	go t.start()
	return &t
}

// start starts the wait timer. Once it expires, `methodToCall` is called. The timer
// can be cancelled by calling `canel()`.
func (t *waitTimer) start() {
	// Update the message every five seconds with the new expiration time until the
	// time has expired.
	for !time.Now().After(t.expiration) {
		maximumWait := time.Until(t.expiration)
		timeToWait := math.Min(maximumWait, 5*time.Second)
		if timeToWait < 0 {
			break
		}
		wait := time.NewTimer(timeToWait)
		select {
		case <-wait.C:
			err := heistMessage(t.s, t.i, "update")
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

// cancel disables the wait timer.
func (t *waitTimer) cancel() {
	log.Trace("--> timer.cancel")
	defer log.Trace("<-- timer.cancel")
	t.timerChannel <- 1
}
