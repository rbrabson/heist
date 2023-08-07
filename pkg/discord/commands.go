package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	helpCommandHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help":      help,
		"adminhelp": adminHelp,
		"version":   version,
	}

	helpCommands = []*discordgo.ApplicationCommand{
		{

			Name:        "help",
			Description: "Provides a description of commands for this server.",
		},
		{
			Name:        "adminhelp",
			Description: "Provides a description of admin commands for this server.",
		},
		{
			Name:        "version",
			Description: "Returns the version of heist running on the server.",
		},
	}
)

// help sends a help message for player commands.
func help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> help")
	log.Debug("<-- help")

	msg.SendEphemeralResponse(s, i, getMemberHelp())
}

// adminHelp sends a help message for administrative commands.
func adminHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> adminHelp")
	log.Debug("<-- adminHelp")

	msg.SendResponse(s, i, getAdminHelp())
}

// version shows the version of heist you are running.
func version(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> version")
	defer log.Debug("<-- version")

	msg.SendEphemeralResponse(s, i, "You are running Heist version "+botVersion+".")
}
