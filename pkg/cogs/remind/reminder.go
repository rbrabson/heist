package remind

import (
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/format"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
)

const (
	REMINDER = "reminder"
)

var (
	servers map[string]*server
)

// Server represents the server on which members may create reminders
type server struct {
	ID      string                   `json:"_id" bson:"_id"`
	Members map[string]*reminderList `json:"members" bson:"members"`
}

// reminderList is a set of reminders for a given member
type reminderList struct {
	MemberID  string      `json:"_id" bson:"_id"`
	Reminders []*reminder `json:"reminders,omitempty" bson:"reminders,omitempty"`
}

// reminder is a reminder for a given member.
type reminder struct {
	Duration time.Duration `json:"duration" bson:"duration"`
	When     time.Time     `json:"when" bson:"when"`
	Message  *string       `json:"message,omitempty" bson:"message,omitempty"`
	Channel  string        `json:"channel" bson:"channel"`
}

// init initializes the set of reminders
func init() {
	servers = make(map[string]*server)
}

// getServer returns the given server. If necessary, a new one is created.
func getServer(serverID string) *server {
	s, ok := servers[serverID]
	if !ok {
		memberList := make(map[string]*reminderList)
		s = &server{
			ID:      serverID,
			Members: memberList,
		}
		servers[s.ID] = s
	}

	return s
}

// newReminder creates a new reminder and adds it to the set of reminders for a given member.
func (s *server) newReminder(channelID string, memberID string, wait time.Duration, message ...string) {
	rl, ok := s.Members[memberID]
	if !ok {
		reminders := make([]*reminder, 0, 1)
		rl = &reminderList{
			MemberID:  memberID,
			Reminders: reminders,
		}
		s.Members[rl.MemberID] = rl
	}

	// There is at most one message, so save it if it is present
	var msg *string
	if len(message) != 0 {
		msg = &message[0]
	}
	channel := fmt.Sprintf("https://discord.com/channels/%s/%s", s.ID, channelID)
	r := &reminder{
		Duration: wait,
		When:     time.Now().Add(wait),
		Message:  msg,
		Channel:  channel,
	}
	rl.Reminders = append(rl.Reminders, r)

	sort.Slice(rl.Reminders, func(i, j int) bool {
		return rl.Reminders[i].When.Before(rl.Reminders[j].When)
	})
}

// createReminder sets a reminder for a person that will be sent via a Direct Message once the
// timer expires.
func (s *server) createReminder(channelID string, memberID string, when string, message ...string) (string, error) {
	log.Trace("--> createReminder")
	defer log.Trace("<-- createReminder")

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

	s.newReminder(channelID, memberID, wait, message...)
	saveReminders(s)

	msg := fmt.Sprintf("I will remind you of that in %s", format.Duration(wait))
	return msg, nil
}

// getReminders returns the list of upcoming reminders for the user.
func getReminders(serverID string, memberID string) (string, error) {
	log.Trace("--> getReminders")
	defer log.Trace("<-- getReminders")

	s := getServer(serverID)
	reminders, ok := s.Members[memberID]
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

// deleteReminders deletes all reminders for the member.
func deleteReminders(serverID string, memberID string) (string, error) {
	log.Trace("--> deleteReminders")
	defer log.Trace("<-- deleteReminders")

	s := getServer(serverID)
	if _, ok := s.Members[memberID]; !ok {
		return "You don't have any upcoming notifications.", ErrNoReminders
	}
	delete(s.Members, memberID)
	saveReminders(s)
	return "All your notifications have been removed.", nil

}

// sendReminders sends a reminder to players who have set a reminder whose wait duration has expired.
func sendReminders() {
	for {
		time.Sleep(15 * time.Second)
		now := time.Now()
		for _, s := range servers {
			saveServer := false
			delIDs := make([]string, 0, 1)
			for _, member := range s.Members {
				for len(member.Reminders) > 0 {
					reminder := member.Reminders[0]
					if reminder.When.After(now) {
						break
					}

					c, err := session.UserChannelCreate(member.MemberID)
					if err != nil {
						log.Errorf("Error creating private channel to %s, error=%s", member.MemberID, err.Error())
						break
					}
					var embed *discordgo.MessageEmbed
					desc := fmt.Sprintf("From %s ago:\n\n%s", format.Duration(reminder.Duration), reminder.Channel)
					if reminder.Message == nil {
						embed = &discordgo.MessageEmbed{
							Type:        discordgo.EmbedTypeRich,
							Title:       ":bell: Reminder! :bell:",
							Description: desc,
						}
					} else {
						embed = &discordgo.MessageEmbed{
							Type:        discordgo.EmbedTypeRich,
							Title:       ":bell: Reminder! :bell:",
							Description: desc,
							Fields: []*discordgo.MessageEmbedField{
								{
									Name:   "Message",
									Value:  *reminder.Message,
									Inline: true,
								},
							},
						}
					}
					_, err = session.ChannelMessageSendEmbed(c.ID, embed)
					if err != nil {
						log.Errorf("Failed to send DM, message=%s", err.Error())
						break
					}

					member.Reminders = member.Reminders[1:]
					saveServer = true
				}
				if len(member.Reminders) == 0 {
					delIDs = append(delIDs, member.MemberID)
				}
			}
			for _, delID := range delIDs {
				delete(s.Members, delID)
			}
			if saveServer {
				saveReminders(s)
			}
		}
	}
}

// loadReminders loads reminders for all members.
func loadReminders() {
	log.Trace("--> LoadReminders")
	defer log.Trace("<-- LoadReminders")

	servers := make(map[string]*server)
	serverIDs := store.Store.ListDocuments(REMINDER)
	for _, serverID := range serverIDs {
		var server server
		store.Store.Load(REMINDER, serverID, &server)
		log.Debug("Server:", server)
		servers[server.ID] = &server
	}
}

// saveReminders saves the reminders for a member.
func saveReminders(server *server) {
	log.Trace("--> SaveReminder")
	defer log.Trace("<-- SaveReminder")

	store.Store.Save(REMINDER, server.ID, server)
}

// GetMemberHelp returns help information about the heist bot commands
func GetMemberHelp() []string {
	help := make([]string, 0, 1)

	for _, command := range commands {
		commandDescription := fmt.Sprintf("- **/%s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Reminder**\n"}, help...)

	return help
}

// GetAdminHelp returns help information about the heist bot commands
func GetAdminHelp() []string {
	return nil
}

// String returns a string repesentation for all reminders on the given server.
func (s *server) String() string {
	out, _ := json.Marshal(s)
	return string(out)
}

// String returns a string representation for all reminders for a given member.
func (r *reminderList) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}

// String returns a string representation of the reminder
func (r *reminder) String() string {
	out, _ := json.Marshal(r)
	return string(out)
}
