package discord

import (
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/rbrabson/heist/pkg/economy"
	"github.com/rbrabson/heist/pkg/heist"
	"github.com/rbrabson/heist/pkg/payday"
	"github.com/rbrabson/heist/pkg/remind"
	log "github.com/sirupsen/logrus"
)

const (
	botIntents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentDirectMessages
)

type Bot struct {
	Session *discordgo.Session
	timer   chan int
}

func addCommands(componentHandlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate),
	commandHandlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate),
	commands []*discordgo.ApplicationCommand,
	getCommands func() (map[string]func(*discordgo.Session, *discordgo.InteractionCreate),
		map[string]func(*discordgo.Session, *discordgo.InteractionCreate),
		[]*discordgo.ApplicationCommand)) []*discordgo.ApplicationCommand {

	compHandlers, cmdHandlers, cmds := getCommands()
	for k, handler := range compHandlers {
		componentHandlers[k] = handler
	}
	for k, handler := range cmdHandlers {
		commandHandlers[k] = handler
	}
	if cmds != nil {
		commands = append(commands, cmds...)
	}
	return commands
}

// NewBot creates a new Discord bot that can run commands for various services.
func NewBot() *Bot {
	godotenv.Load()
	guildID := os.Getenv("HEIST_GUILD_ID")
	appID := os.Getenv("APP_ID")

	token := os.Getenv("BOT_TOKEN")
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		log.Fatal("Failed to create new bot, error:", err)
	}

	bot := &Bot{
		Session: s,
		timer:   make(chan int),
	}
	bot.Session.Identify.Intents = botIntents

	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Heist bot is up!")
	})

	economy.LoadBanks()

	heist.Start(bot.Session)
	payday.Start(bot.Session)
	remind.Start(bot.Session)

	componentHandlers := make(map[string]func(*discordgo.Session, *discordgo.InteractionCreate))
	commandHandlers := make(map[string]func(*discordgo.Session, *discordgo.InteractionCreate))
	commands := make([]*discordgo.ApplicationCommand, 0)
	commands = addCommands(componentHandlers, commandHandlers, commands, heist.GetCommands)
	commands = addCommands(componentHandlers, commandHandlers, commands, payday.GetCommands)
	commands = addCommands(componentHandlers, commandHandlers, commands, remind.GetCommands)

	log.Debug("Add bot handlers")
	bot.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			if h, ok := componentHandlers[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}
		}
	})

	/*
		// Delete any old slash commands, and then add in my current set
		log.Info("Delete old commands")
		_, err := s.ApplicationCommandBulkOverwrite(appID, guildID, nil)
		if err != nil {
			log.Fatal("Failed to delete all old commands, error:", err)
		}
	*/

	log.Debug("Add bot commands")
	_, err = bot.Session.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	if err != nil {
		log.Fatal("Failed to load heist commands, error:", err)
	}

	return bot
}
