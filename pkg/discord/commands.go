package discord

import (
	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	helpCommandHandler = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"help": help,
	}

	helpCommand = &discordgo.ApplicationCommand{
		Name:        "help",
		Description: "Provides a description of commands for this server",
	}
)

func help(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> help")
	log.Debug("<-- help")

	msg.SendResponse(s, i, getMemberHelp())
}
