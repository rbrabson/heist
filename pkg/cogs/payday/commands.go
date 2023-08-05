package payday

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/format"
	discmsg "github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"payday": payday,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "payday",
			Description: "Deposits your daily check into your bank account.",
		},
	}
)

var (
	servers map[string]*server
)

// payday gives some credits to the player every 24 hours.
func payday(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> payday")
	defer log.Debug("<-- payday")

	p := getPrinter(i)
	server := getServer(i.GuildID)
	member := server.getMember(i.Member.User.ID)

	if member.NextPayday.After(time.Now()) {
		remainingTime := time.Until(member.NextPayday)
		msg := p.Sprintf("You can't get another payday yet. You need to wait %s.", format.Duration(remainingTime))
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	bank := economy.GetBank(i.GuildID)
	account := bank.GetAccount(i.Member.User.ID, i.Member.User.Username) // TODO: handle the nickname
	economy.DepositCredits(bank, account, int(server.PaydayAmount))
	economy.SaveBank(bank)
	member.NextPayday = time.Now().Add(server.PaydayFrequency)
	saveServer(server)

	msg := p.Sprintf("You deposited your check of %d into your bank account. You now have %d credits.", server.PaydayAmount, account.Balance)
	discmsg.SendEphemeralResponse(s, i, msg)
}

// Start initializes the payday information.
func Start(s *discordgo.Session) {
	servers = loadServers()
}

// GetCommands returns the component handlers, command handlers, and commands for the payday bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, commandHandlers, commands
}
