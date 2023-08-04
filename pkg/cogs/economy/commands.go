package economy

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"bank": bank,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "bank",
			Description: "Commands used to interact with the economy for this server.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "set",
					Description: "Sets the amount of credits for a given member.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "id",
							Description: "The member ID.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "amount",
							Description: "The amount to set the account to.",
							Required:    true,
						},
					},
				},
				{
					Name:        "transfer",
					Description: "Transfers all credits from one account to another.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "from",
							Description: "The ID of the member to transfer credits from.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "to",
							Description: "The ID of the member to transfer credits to.",
							Required:    true,
						},
					},
				},
			},
		},
	}
)

// bank routes the bank commands to the proper handers.
func bank(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> bank")
	defer log.Debug("<-- bank")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "set":
		setAccount(s, i)
	case "transfer":
		transferAccount(s, i)
	}
}

// setAccount sets the account to the specified number of credits.
func setAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> setAccount")
	defer log.Debug("<-- setAccount")

	var id string
	var amount int64
	options := i.ApplicationCommandData().Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "id":
			id = strings.TrimSpace(option.StringValue())
		case "amount":
			amount = option.IntValue()
		}
	}

	p := getPrinter(i)

	member, err := s.GuildMember(i.GuildID, id)
	if err != nil {
		resp := p.Sprintf("A account with ID `%s` is not a member of this server", id)
		msg.SendEphemeralResponse(s, i, resp)
		return
	}

	bank := GetBank(i.GuildID)
	account := bank.GetAccount(id, getMemberName(member.User.ID, member.Nick))
	account.Balance = int(amount)
	SaveBank(bank)

	resp := p.Sprintf("Account for %s was set to %d credits.", account.Name, account.Balance)
	msg.SendResponse(s, i, resp)
}

// transferAccount sets the target account to the amount of credits in the source
// account, and clears the account balance of the source.
func transferAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> transferAccount")
	defer log.Debug("<-- transferAccount")

	var fromID, toID string
	options := i.ApplicationCommandData().Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "from":
			fromID = strings.TrimSpace(option.StringValue())
		case "to":
			toID = strings.TrimSpace(option.StringValue())
		}
	}

	p := getPrinter(i)

	bank := GetBank(i.GuildID)
	fromAccount, ok := bank.Accounts[fromID]
	if !ok {
		resp := p.Sprintf("Account %s does not exist.")
		msg.SendEphemeralResponse(s, i, resp)
		return
	}

	member, err := s.GuildMember(i.GuildID, toID)
	if err != nil {
		resp := p.Sprintf("An account with ID `%s` is not a member of this server", toID)
		msg.SendEphemeralResponse(s, i, resp)
		return
	}

	toAccount := bank.GetAccount(toID, getMemberName(member.User.Username, member.Nick))

	toAccount.Balance = fromAccount.Balance
	fromAccount.Balance = 0

	SaveBank(bank)

	resp := p.Sprintf("Transferred balance of %d from %s to %s.", toAccount.Balance, fromAccount.Name, toAccount.Name)
	msg.SendResponse(s, i, resp)

}

// Start intializes the economy.
func Start(session *discordgo.Session) {
	LoadBanks()
}

// GetCommands returns the component handlers, command handlers, and commands for the payday bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, commandHandlers, commands
}
