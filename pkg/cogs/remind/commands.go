package remind

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	session *discordgo.Session
)

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"remadd":  addReminder,
		"remdel":  removeReminders,
		"remlist": listReminders,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "remadd",
			Description: "Sets a new reminder for you.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "when",
					Description: "Time to wait before sending the reminder",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "message",
					Description: "Message to send with the reminder",
					Required:    false,
				},
			},
		},
		{
			Name:        "remdel",
			Description: "Removes all of your reminders.",
		},
		{
			Name:        "remlist",
			Description: "List all of your reminders.",
		},
	}
)

// addReminder adds a new reminder for the member.
func addReminder(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> addReminder")
	defer log.Debug("<-- addReminder")

	var when, message string
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "when":
			when = option.StringValue()
		case "message":
			message = option.StringValue()
		}
	}

	var response string
	if message == "" {
		log.WithFields(log.Fields{
			"GuildID":  i.GuildID,
			"MemberID": i.Member.User.ID,
			"When":     when,
		}).Debug("Creating a reminder")
		response, _ = createReminder(i.GuildID, i.Member.User.ID, when)
	} else {
		log.WithFields(log.Fields{
			"GuildID":  i.GuildID,
			"MemberID": i.Member.User.ID,
			"When":     when,
			"Message":  message,
		}).Debug("Creating a reminder")
		response, _ = createReminder(i.GuildID, i.Member.User.ID, when, message)
	}

	msg.SendEphemeralResponse(s, i, response)
}

// listReminders returns a list of all reminders for the member.
func listReminders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listReminders")
	defer log.Debug("<-- listReminders")

	response, _ := getReminders(i.GuildID, i.Member.User.ID)
	msg.SendEphemeralResponse(s, i, response)
}

// removeReminders deletes all reminders for the member.
func removeReminders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> removeReminders")
	defer log.Debug("<-- removeReminders")

	response, _ := deleteReminders(i.GuildID, i.Member.User.ID)
	msg.SendEphemeralResponse(s, i, response)

}

// GetCommands returns the component handlers, command handlers, and commands for the remind bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, commandHandlers, commands
}

// Start starts up the bot
func Start(s *discordgo.Session) {
	session = s
	loadReminders()
	go sendReminders()
}
