package economy

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
	ECONOMY = "economy"
)

var (
	banks map[string]*Bank
)

// BankStore defines the methods required to load and save the economy state.
type BankStore interface {
	loadBanks() map[string]*Bank
	saveBank(*Bank)
}

// Bank is the repository for all accounts for a given server/guild.
type Bank struct {
	ID                  string              `json:"_id" bson:"_id"`
	BankName            string              `json:"bank_name" bson:"bank_name"`
	Currency            string              `json:"currency" bson:"currency"`
	DefaultBalance      int                 `json:"default_balance" bson:"default_balance"`
	Accounts            map[string]*Account `json:"accounts" bson:"accounts"`
	MaxTransferAmount   int                 `json:"max_transfer_amount" bson:"max_transfer_amount"`
	MinTransferDuration time.Duration       `json:"min_transfer_duration" bson:"min_transfer_duration"`
}

// Account is the bank account for a member of the server/guild.
type Account struct {
	ID              string    `json:"_id" bson:"_id"`
	Balance         int       `json:"balance" bson:"balance"`
	CreatedAt       time.Time `json:"created_at" bson:"created_at"`
	Name            string    `json:"name" bson:"name"`
	NextTransferIn  time.Time `json:"next_transfer_in" bson:"next_transfer_in"`
	NextTransferOut time.Time `json:"next_transfer_out" bson:"next_transfer_out"`
}

// NewBank creates a new bank for the given server/guild.
func NewBank(serverID string) *Bank {
	bank := Bank{
		ID:                  serverID,
		DefaultBalance:      0,
		BankName:            "Treasury",
		Currency:            "Coins",
		MaxTransferAmount:   50000,
		MinTransferDuration: time.Duration(24 * time.Hour),
	}
	bank.Accounts = make(map[string]*Account)
	return &bank
}

// GetBank returns the bank for the server/guild.
func GetBank(serverID string) *Bank {
	bank, ok := banks[serverID]
	if !ok {
		bank = NewBank(serverID)
		log.Println(len(banks), bank)
		banks[bank.ID] = bank
		log.Warningf("Bank not found for server %s, new one created", serverID)
	}
	return bank
}

// NewAccount creates a new bank account for the player.
func NewAccount(b *Bank, playerID string, playerName string) *Account {
	account := Account{
		ID:        playerID,
		Balance:   b.DefaultBalance,
		CreatedAt: time.Now(),
		Name:      playerName,
	}
	return &account
}

// GetAccount returns the bank account for the player.
func (b *Bank) GetAccount(playerID string, playerName string) *Account {
	account, ok := b.Accounts[playerID]
	if !ok {
		account = NewAccount(b, playerID, playerName)
		b.Accounts[account.ID] = account
		log.Error("Account for " + playerName + " was not found")
	}

	return account
}

// DepositCredits adds the amount of credits to the account at a given bank
func DepositCredits(bank *Bank, account *Account, amount int) {
	account.Balance += amount
}

// WithDrawCredits deducts the amount of credits from the account at the given bank
func WithdrawCredits(bank *Bank, account *Account, amount int) error {
	if account.Balance < amount {
		return ErrInsufficintBalance
	}
	account.Balance -= amount
	return nil
}

// LoadBanks returns all the banks for the given guilds.
func LoadBanks() {
	banks = make(map[string]*Bank)
	bankIDs := store.Store.ListDocuments(ECONOMY)
	for _, bankID := range bankIDs {
		var bank Bank
		store.Store.Load(ECONOMY, bankID, &bank)
		banks[bank.ID] = &bank
	}
}

// SaveBank saves the bank.
func SaveBank(bank *Bank) {
	store.Store.Save(ECONOMY, bank.ID, bank)
}

// getMemberName returns the member's nickname, if there is one, or the username otherwise.
func getMemberName(username string, nickname string) string {
	if nickname != "" {
		return nickname
	}
	return username
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

// GetHelp returns help information about the heist bot commands
func GetMemberHelp() []string {
	help := make([]string, 0, len(memberCommands))

	for _, command := range memberCommands {
		commandDescription := fmt.Sprintf("- **/%s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Economy**\n"}, help...)

	return help
}

// GetAdminHelp returns help information about the heist bot commands
func GetAdminHelp() []string {
	help := make([]string, 0, len(adminCommands))

	for _, command := range adminCommands {
		commandDescription := fmt.Sprintf("- **/%s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Economy**\n"}, help...)

	return help
}
