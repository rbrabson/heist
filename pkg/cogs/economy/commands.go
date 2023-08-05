package economy

import (
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/format"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
)

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"bank":     bank,
		"transfer": transferCredits,
	}

	adminCommands = []*discordgo.ApplicationCommand{
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
					Description: "Transfers the account balance from one account to another.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "from",
							Description: "The ID of the account to transfer credits from.",
							Required:    true,
						},
						{
							Type:        discordgo.ApplicationCommandOptionInteger,
							Name:        "to",
							Description: "The ID of the account to receive account balance.",
							Required:    true,
						},
					},
				},
			},
		},
	}

	memberCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "transfer",
			Description: "Transfers a set amount of credits from your account to another player's account.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "to",
					Description: "The ID of the member to transfer credits to.",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "amount",
					Description: "The amount of credits to transfer.",
					Required:    true,
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

// transferCredits removes a specified amount of credits from initiators account and deposits them in the target's account.
func transferCredits(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> bank")
	defer log.Debug("<-- bank")

	var toID string
	var amount int
	options := i.ApplicationCommandData().Options
	for _, option := range options {
		switch option.Name {
		case "to":
			toID = option.StringValue()
		case "amount":
			amount = int(option.IntValue())
		}
	}

	p := getPrinter(i)

	member, err := s.GuildMember(i.GuildID, toID)
	if err != nil {
		resp := p.Sprintf("A account with ID `%s` is not a member of this server", toID)
		msg.SendEphemeralResponse(s, i, resp)
		return
	}

	bank := GetBank(i.GuildID)
	fromAccount := bank.GetAccount(i.Member.User.ID, getMemberName(i.Member.User.ID, i.Member.Nick))
	toAccount := bank.GetAccount(toID, getMemberName(member.User.ID, member.Nick))

	if fromAccount.NextTransferOut.After(time.Now()) {
		duration := time.Until(fromAccount.NextTransferOut)
		resp := p.Sprintf("You can't transfer credits yet. Please wait %s to transfer credits.", format.Duration(duration))
		msg.SendEphemeralResponse(s, i, resp)
		return
	}
	if toAccount.NextTransferIn.After(time.Now()) {
		duration := time.Until(toAccount.NextTransferIn)
		resp := p.Sprintf("%s can't receive credits yet. Please wait %s to transfer credits.", format.Duration(duration))
		msg.SendEphemeralResponse(s, i, resp)
		return
	}
	if amount > bank.MaxTransferAmount {
		resp := p.Sprintf("You can only transfer a maximum of %d credits.", bank.MaxTransferAmount)
		msg.SendEphemeralResponse(s, i, resp)
		return
	}
	if fromAccount.Balance < amount {
		resp := p.Sprintf("Your can't transfer %d credits as your account only has %d credits", amount, fromAccount.Balance)
		msg.SendEphemeralResponse(s, i, resp)
		return
	}

	log.WithFields(log.Fields{
		"CurrentTime":     time.Now(),
		"NextTransferOut": fromAccount.NextTransferOut,
		"NextTransferIn":  toAccount.NextTransferIn,
		"Interval":        bank.MinTransferDuration,
	}).Info("/transfer")

	fromAccount.Balance -= amount
	toAccount.Balance += amount
	fromAccount.NextTransferOut = time.Now().Add(bank.MinTransferDuration)
	toAccount.NextTransferIn = time.Now().Add(bank.MinTransferDuration)
	SaveBank(bank)
	resp := p.Sprintf("You transfered %d credits to %s's account.", amount, toAccount.Name)
	msg.SendResponse(s, i, resp)
}

// setAccount sets the account to the specified number of credits.
func setAccount(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> setAccount")
	defer log.Debug("<-- setAccount")

	var id string
	var amount int
	options := i.ApplicationCommandData().Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "id":
			id = strings.TrimSpace(option.StringValue())
		case "amount":
			amount = int(option.IntValue())
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
	account.Balance = amount
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
	commands := make([]*discordgo.ApplicationCommand, 0, len(memberCommands)+len(adminCommands))
	commands = append(commands, memberCommands...)
	commands = append(commands, adminCommands...)
	return nil, commandHandlers, commands
}
