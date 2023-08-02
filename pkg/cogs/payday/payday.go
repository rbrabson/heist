package payday

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/economy"
	"github.com/rbrabson/heist/pkg/format"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"

	discmsg "github.com/rbrabson/heist/pkg/msg"
)

const (
	PAYDAY = "payday"
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
	servers map[string]*server
)

// server is the server/guild on which the payday checks are deposited
type server struct {
	ID              string             `json:"_id" bson:"_id"`
	Members         map[string]*member `json:"members" bson:"members"`
	PaydayAmount    int64              `json:"payday_amount" bson:"payday_amount"`
	PaydayFrequency time.Duration      `json:"payday_frequency" bson:"payday_frequency"`
}

// member is the member of the server/guild who deposits the payday check.
type member struct {
	ID         string    `json:"_id" bson:"_id"`
	NextPayday time.Time `json:"next_payday" bson:"next_payday"`
}

func init() {
	servers = make(map[string]*server)
}

// newServer creates a new server/guild
func newServer(serverID string) *server {
	members := make(map[string]*member)
	server := &server{
		ID:              serverID,
		Members:         members,
		PaydayAmount:    5000,
		PaydayFrequency: time.Duration(24 * time.Hour),
	}
	servers[server.ID] = server
	saveServer(server)

	return server
}

// getServer returns the server/guild, creating a new one if necessary.
func getServer(serverID string) *server {
	server, ok := servers[serverID]
	if !ok {
		server = newServer(serverID)
	}

	return server
}

// newMember creates a new member of a guild.
func (s *server) newMember(memberID string) *member {
	member := &member{
		ID:         memberID,
		NextPayday: time.Now(),
	}
	s.Members[member.ID] = member
	saveServer(s)
	return member
}

// getMember gets the member of the server/guild, creating a new one if necessary.
func (s *server) getMember(memberID string) *member {
	member, ok := s.Members[memberID]
	if !ok {
		member = s.newMember(memberID)
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

// GetPaydayAmount returns the amount of credits a player depsots into their account on a given payday.
func GetPaydayAmount(serverID string) int64 {
	log.Debug("--> GetPaydayAmount")
	defer log.Debug("<-- GetPaydayAmount")

	server := getServer(serverID)
	return server.PaydayAmount
}

// SetPaydayAmount sets the amount of credits a player deposits into their account on a given payday.
func SetPaydayAmount(serverID string, amount int64) {
	log.Debug("--> SetPaydayAmount")
	defer log.Debug("<-- SetPaydayAmount")

	server := getServer(serverID)
	server.PaydayAmount = amount

	saveServer(server)
}

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

// loadServers loads payday information for all servers from the store.
func loadServers() map[string]*server {
	log.Debug("--> loadServers")
	defer log.Debug("<-- loadServers")

	servers := make(map[string]*server)
	serverIDs := store.Store.ListDocuments(PAYDAY)
	for _, serverID := range serverIDs {
		var server server
		store.Store.Load(PAYDAY, serverID, &server)
		servers[server.ID] = &server
	}

	return servers
}

// saveServer saves the payday information for the server into the store.
func saveServer(server *server) {
	log.Debug("--> saveServer")
	defer log.Debug("<-- saveServer")

	store.Store.Save(PAYDAY, server.ID, server)
}

// Start initializes the payday information.
func Start(s *discordgo.Session) {
	servers = loadServers()
}

// GetCommands returns the component handlers, command handlers, and commands for the payday bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, commandHandlers, commands
}
