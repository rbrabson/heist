package remind

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rbrabson/heist/pkg/format"
	log "github.com/sirupsen/logrus"
)

var (
	state map[string]*reminderList
)

// reminderList is a set of reminders for a given member
type reminderList struct {
	MemberID  string      `json:"_id" bson:"_id"`
	Reminders []*reminder `json:"reminders,omitempty" bson:"reminders,omitempty"`
}

// reminder is a reminder for a given member.
type reminder struct {
	Duration time.Duration `json:"duragion" bson:"duration"`
	When     time.Time     `json:"when" bson:"when"`
	Message  *string       `json:"message,omitempty" bson:"message,omitempty"`
}

// init initializes the set of reminders
func init() {
	state = make(map[string]*reminderList)
}

// newReminder creates a new reminder and adds it to the set of reminders for a given member.
func newReminder(memberID string, wait time.Duration, message ...string) {
	rl, ok := state[memberID]
	if !ok {
		reminders := make([]*reminder, 0, 1)
		rl = &reminderList{
			MemberID:  memberID,
			Reminders: reminders,
		}
		state[rl.MemberID] = rl
	}

	// There is at most one message, so save it if it is present
	var msg *string
	if len(message) != 0 {
		msg = &message[0]
	}
	r := &reminder{
		Duration: wait,
		When:     time.Now().Add(wait),
		Message:  msg,
	}
	rl.Reminders = append(rl.Reminders, r)

	sort.Slice(rl.Reminders, func(i, j int) bool {
		return rl.Reminders[i].When.Before(rl.Reminders[j].When)
	})
}

// CreateReminder sets a reminder for a person that will be sent via a Direct Message once the
// timer expires.
func CreateReminder(memberID string, when string, message ...string) (string, error) {
	log.Debug("--> CreateReminder")
	defer log.Debug("<-- CreateReminder")

	// If the time is just a number, default to hours
	_, err := strconv.Atoi(when)
	if err == nil {
		when += "h"
	}

	wait, err := time.ParseDuration(when)
	if err != nil {
		msg := fmt.Sprintf("Unable to parse duration of %s", when)
		return msg, ErrInvalidDuration
	}

	newReminder(memberID, wait, message...)

	msg := fmt.Sprintf("I will remind you of that in %s", format.Duration(wait))
	return msg, nil
}

// ListReminders returns the list of upcoming reminders for the user.
func ListReminders(memberID string) (string, error) {
	log.Debug("--> ListReminders")
	defer log.Debug("<-- ListReminders")

	reminders, ok := state[memberID]
	if !ok {
		msg := "You don't have any upcoming notifications"
		return msg, ErrNoReminders
	}
	var sb strings.Builder
	for _, reminder := range reminders.Reminders {
		wait := time.Until(reminder.When)
		var msg string
		if reminder.Message == nil {
			msg = fmt.Sprintf("You asked me to remind you in %s\n", format.Duration(wait))
		} else {
			msg = fmt.Sprintf("You asked me to remind you of this in %s: \"%s\"\n", format.Duration(wait), *reminder.Message)
		}
		sb.WriteString(msg)
	}

	return sb.String(), nil
}

// ForgetReminders deletes all reminders for the member.
func ForgetReminders(memberID string) (string, error) {
	log.Debug("--> ForgetReminders")
	defer log.Debug("<-- ForgetReminders")

	if _, ok := state[memberID]; !ok {
		return "You don't have any upcoming notifications.", ErrNoReminders
	}
	delete(state, memberID)
	return "All your notifications have been removed.", nil
}

// sendReminders sends a reminder to players who have set a reminder whose wait duration has expired.
func sendReminders() {
	for {
		time.Sleep(15 * time.Second)
		now := time.Now()
		for _, rl := range state {
			for len(rl.Reminders) > 0 {
				reminder := rl.Reminders[0]
				if reminder.When.After(now) {
					break
				}

				c, err := session.UserChannelCreate(rl.MemberID)
				if err != nil {
					log.Errorf("Error creating private channel to %s, error=%s", rl.MemberID, err.Error())
					break
				}

				var message string
				if reminder.Message == nil {
					message = fmt.Sprintf(":bell: Reminder! :bell:\nFrom %s ago", format.Duration(reminder.Duration))
				} else {
					message = fmt.Sprintf(":bell: Reminder! :bell:\nFrom %s ago\n\n%s", format.Duration(reminder.Duration), *reminder.Message)
				}
				_, err = session.ChannelMessageSend(c.ID, message)
				if err != nil {
					log.Errorf("Failed to send DM, message=%s", err.Error())
					break
				} else {
					log.Debugf("Sent DM to %s\nmessage=%s", rl.MemberID, message)
				}

				rl.Reminders = rl.Reminders[1:]
			}
		}
	}
}
