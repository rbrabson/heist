package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/rbrabson/heist/pkg/discord"
	log "github.com/sirupsen/logrus"
)

func main() {
	godotenv.Load()

	bot := discord.NewBot()
	err := bot.Session.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Session.Close()

	// log.SetLevel(log.DebugLevel)
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Info("Press Ctrl+C to exit")
	<-sc
}
