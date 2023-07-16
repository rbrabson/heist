package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/rbrabson/heist/pkg/heist"
	log "github.com/sirupsen/logrus"
)

func main() {
	bot := heist.NewBot()

	err := bot.Session.Open()
	if err != nil {
		log.Fatal(err)
	}
	defer bot.Session.Close()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	log.Info("Press Ctrl+C to exit")
	<-sc
}
