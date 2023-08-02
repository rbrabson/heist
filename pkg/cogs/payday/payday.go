package payday

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/economy"
	"github.com/rbrabson/heist/pkg/format"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	discmsg "github.com/rbrabson/heist/pkg/msg"
)

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"payday": payday,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "payday",
			Description: "Deposits your daily check into your bank account",
		},
	}
)

var (
	PaydayAmount = 5000
)

var (
	servers map[string]*Server
)

// Server is the server/guild on which the payday checks are deposited
type Server struct {
	ID      string             `json:"_id" bson:"_id"`
	Members map[string]*Member `json:"members" bson:"members"`
}

// Member is the member of the server/guild who deposits the payday check.
type Member struct {
	ID         string    `json:"_id" bson:"_id"`
	NextPayday time.Time `json:"next_payday" bson:"next_payday"`
}

func init() {
	servers = make(map[string]*Server)
}

// newServer creates a new server/guild
func newServer(serverID string) *Server {
	members := make(map[string]*Member)
	server := &Server{
		ID:      serverID,
		Members: members,
	}
	servers[server.ID] = server
	return server
}

// getServer returns the server/guild, creating a new one if necessary.
func getServer(serverID string) *Server {
	server, ok := servers[serverID]
	if !ok {
		server = newServer(serverID)
	}
	return server
}

// newMember creates a new member of a guild.
func newMember(serverID string, memberID string) *Member {
	server := getServer(serverID)
	member := &Member{
		ID:         memberID,
		NextPayday: time.Now(),
	}
	server.Members[member.ID] = member
	return member
}

// getMember gets the member of the server/guild, creating a new one if necessary.
func getMember(serverID string, memberID string) *Member {
	server := getServer(serverID)
	member, ok := server.Members[memberID]
	if !ok {
		member = newMember(serverID, memberID)
	}
	return member
}

// getPrinter returns a printer for the given locale of the user initiating the message.
func getPrinter(i *discordgo.InteractionCreate) *message.Printer {
	tag, err := language.Parse(string(i.Locale))
	if err != nil {
		log.Error("Unable to parse locale, error:", err)
		tag = language.English
	}
	return message.NewPrinter(tag)
}

// payday gives some credits to the player every 24 hours.
func payday(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> payday")
	defer log.Debug("<-- payday")

	p := getPrinter(i)

	payday := getMember(i.GuildID, i.Member.User.ID)
	if payday.NextPayday.After(time.Now()) {
		remainingTime := time.Until(payday.NextPayday)
		msg := p.Sprintf("You can't get another payday yet. You need to wait %s.", format.Duration(remainingTime))
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	bank := economy.GetBank(i.GuildID)
	account := bank.GetAccount(i.Member.User.ID, i.Member.User.Username) // TODO: handle the nickname
	economy.DepositCredits(bank, account, PaydayAmount)
	payday.NextPayday = time.Now().Add(24 * time.Hour)
	economy.SaveBank(bank)

	msg := p.Sprintf("You deposited your check of %d into your bank account. You now have %d credits.", PaydayAmount, account.Balance)
	discmsg.SendEphemeralResponse(s, i, msg)
}

func Start(s *discordgo.Session) {
	// no-op
}

// GetCommands returns the component handlers, command handlers, and commands for the payday bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, commandHandlers, commands
}
