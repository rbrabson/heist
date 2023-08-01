package heist

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	timer   chan int
}

func NewBot() *Bot {
	token := os.Getenv("BOT_TOKEN")
	s, _ := discordgo.New("Bot " + token)

	bot := &Bot{
		Session: s,
		timer:   make(chan int),
	}
	bot.Session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged
	addBotCommands(bot)

	log.Debug(servers)

	go bot.vaultUpdater()

	return bot
}

type number interface {
	int | int32 | int64 | float32 | float64 | time.Duration
}

func min[N number](v1 N, v2 N) N {
	if v1 < v2 {
		return v1
	}
	return v2
}

func (b *Bot) vaultUpdater() {
	const timer = time.Duration(120 * time.Second)
	time.Sleep(20 * time.Second)
	for {
		for _, server := range servers {
			for _, target := range server.Targets {
				vault := min(target.Vault+(target.VaultMax*4/100), target.VaultMax)
				target.Vault = vault
			}
		}
		time.Sleep(timer)
	}
}
