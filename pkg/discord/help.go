package discord

import (
	"strings"

	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/cogs/heist"
	"github.com/rbrabson/heist/pkg/cogs/payday"
	"github.com/rbrabson/heist/pkg/cogs/remind"
	log "github.com/sirupsen/logrus"
)

// getHelp gets help about player commands from all bots.
func getMemberHelp() string {
	log.Trace("--> getMemberHelp")
	log.Trace("<-- getMemberHelp")

	var sb strings.Builder

	for _, str := range economy.GetMemberHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range heist.GetMemberHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range payday.GetMemberHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range remind.GetMemberHelp() {
		sb.WriteString(str)
	}

	return sb.String()
}

// getAdminHelp returns help about administrative commands for all bots.
func getAdminHelp() string {
	log.Trace("--> getAdminHelp")
	log.Trace("<-- getAdminHelp")

	var sb strings.Builder

	for _, str := range economy.GetAdminHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range heist.GetAdminHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range payday.GetAdminHelp() {
		sb.WriteString(str)
	}
	sb.WriteString("\n")
	for _, str := range remind.GetAdminHelp() {
		sb.WriteString(str)
	}

	return sb.String()
}
