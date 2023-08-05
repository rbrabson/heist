package payday

import (
	"fmt"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

const (
	PAYDAY = "payday"
)

// server is the server/guild into which payday checks are deposited.
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

// GetMemberHelp returns help information about the heist bot commands
func GetMemberHelp() []string {
	help := make([]string, 0, 1)

	for _, command := range commands {
		commandDescription := fmt.Sprintf("- **/%s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Payday**\n"}, help...)

	return help
}

// GetAdminHelp returns help information about the heist bot commands
func GetAdminHelp() []string {
	return nil
}
