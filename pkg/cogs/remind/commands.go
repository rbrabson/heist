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
		"reminder": reminderRouter,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "reminder",
			Description: "Set, list or delete reminders.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "add",
					Description: "Sets a new reminder for you.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
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
					Name:        "del",
					Description: "Removes all of your reminders.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "list",
					Description: "List all of your reminders.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}
)

// reminderRouter routes the various reminder requests to the appropriate handler.
func reminderRouter(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> reminderRouter")
	defer log.Trace("<-- reminderRouter")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "add":
		addReminder(s, i)
	case "del":
		removeReminders(s, i)
	case "list":
		listReminders(s, i)
	}
}

// addReminder adds a new reminder for the member.
func addReminder(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> addReminder")
	defer log.Trace("<-- addReminder")

	var when, message string
	for _, option := range i.ApplicationCommandData().Options[0].Options {
		switch option.Name {
		case "when":
			when = option.StringValue()
		case "message":
			message = option.StringValue()
		}
	}

	server := getServer(i.GuildID)

	var response string
	if message == "" {
		log.WithFields(log.Fields{
			"GuildID":  i.GuildID,
			"MemberID": i.Member.User.ID,
			"When":     when,
		}).Debug("Creating a reminder")
		response, _ = server.createReminder(i.ChannelID, i.Member.User.ID, when)
	} else {
		log.WithFields(log.Fields{
			"GuildID":  i.GuildID,
			"MemberID": i.Member.User.ID,
			"When":     when,
			"Message":  message,
		}).Debug("Creating a reminder")
		response, _ = server.createReminder(i.ChannelID, i.Member.User.ID, when, message)
	}

	saveReminders(server)
	msg.SendEphemeralResponse(s, i, response)
}

// listReminders returns a list of all reminders for the member.
func listReminders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> listReminders")
	defer log.Trace("<-- listReminders")

	response, _ := getReminders(i.GuildID, i.Member.User.ID)
	msg.SendEphemeralResponse(s, i, response)
}

// removeReminders deletes all reminders for the member.
func removeReminders(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> removeReminders")
	defer log.Trace("<-- removeReminders")

	response, _ := deleteReminders(i.GuildID, i.Member.User.ID)
	server := getServer(i.GuildID)
	saveReminders(server)
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
