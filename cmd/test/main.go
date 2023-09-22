package main

import (
	"log"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
)

const (
	botIntents = discordgo.IntentGuilds |
		discordgo.IntentGuildMessages |
		discordgo.IntentDirectMessages |
		discordgo.IntentGuildEmojis
)

type Bot struct {
	Session *discordgo.Session
	timer   chan int
}

func main() {
	godotenv.Load()

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

	err = bot.Session.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Session.Close()

	channelID := "1133474546121449492"

	embeds := []*discordgo.MessageEmbed{
		{
			Type:  discordgo.EmbedTypeRich,
			Title: "Monthly Leaderboard",
			Fields: []*discordgo.MessageEmbedField{
				{
					Value: "This is where the data would go",
				},
			},
		},
	}
	_, err = bot.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: embeds,
	})
	if err != nil {
		log.Fatal("Unable to send montly leaderboard, err:", err)
	}
}
