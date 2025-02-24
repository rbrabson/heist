package economy

import (
	"fmt"
	"sort"
	"sync"
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
	ID             string              `json:"_id" bson:"_id"`
	BankName       string              `json:"bank_name" bson:"bank_name"`
	Currency       string              `json:"currency" bson:"currency"`
	DefaultBalance int                 `json:"default_balance" bson:"default_balance"`
	Accounts       map[string]*Account `json:"accounts" bson:"accounts"`
	LastSeason     time.Time           `json:"last_season" bson:"last_season"`
	ChannelID      string              `json:"channel_id" bson:"channel_id"`
	mutex          sync.Mutex          `json:"-" bson:"-"`
}

// Account is the bank account for a member of the server/guild.
type Account struct {
	ID              string     `json:"_id" bson:"_id"`
	MonthlyBalance  int        `json:"monthly_balance" bson:"monthly_balance"`
	CurrentBalance  int        `json:"current_balance" bson:"current_balance"`
	LifetimeBalance int        `json:"lifetime_balance" bson:"lifetime_balance"`
	CreatedAt       time.Time  `json:"created_at" bson:"created_at"`
	Name            string     `json:"name" bson:"name"`
	mutex           sync.Mutex `json:"-" bson:"-"`
}

// newBank creates a new bank for the given server/guild.
func newBank(serverID string) *Bank {
	log.Trace("--> NewBank")
	defer log.Trace("<-- NewBank")

	now := time.Now()
	month := now.Month()
	year := now.Year()
	lastMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
	bank := Bank{
		ID:             serverID,
		DefaultBalance: 20000,
		BankName:       "Treasury",
		Currency:       "Coins",
		LastSeason:     lastMonth,
	}
	bank.Accounts = make(map[string]*Account)
	return &bank
}

// GetBank returns the bank for the server/guild.
func GetBank(serverID string) *Bank {
	log.Trace("--> GetBank")
	defer log.Trace("<-- GetBank")

	bank, ok := banks[serverID]
	if !ok {
		bank = newBank(serverID)
		banks[bank.ID] = bank
		log.Warningf("Bank not found for server %s, new one created", serverID)
	}
	return bank
}

// newAccount creates a new bank account for the player.
func newAccount(b *Bank, playerID string, playerName string) *Account {
	log.Trace("--> NewAccount")
	defer log.Trace("<-- NewAccount")

	account := Account{
		ID:              playerID,
		MonthlyBalance:  0,
		CurrentBalance:  b.DefaultBalance,
		LifetimeBalance: b.DefaultBalance,
		CreatedAt:       time.Now(),
		Name:            playerName,
	}
	return &account
}

// GetAccount returns the bank account for the player.
func (b *Bank) GetAccount(playerID string, playerName string) *Account {
	log.Trace("--> GetAccount")
	defer log.Trace("<-- GetAccount")

	b.mutex.Lock()
	defer b.mutex.Unlock()

	account, ok := b.Accounts[playerID]
	if !ok {
		account = newAccount(b, playerID, playerName)
		b.Accounts[account.ID] = account
		log.Warningf("Account for %s was not found, new one created", playerName)
	} else {
		account.Name = playerName
	}

	return account
}

// DepositCredits adds the amount of credits to the account at a given bank
func (a *Account) DepositCredits(amount int) {
	log.Trace("--> DepositCredits")
	defer log.Trace("<-- DepositCredits")

	a.mutex.Lock()
	defer a.mutex.Unlock()

	a.MonthlyBalance += amount
	a.CurrentBalance += amount
	a.LifetimeBalance += amount
}

// WithDrawCredits deducts the amount of credits from the account at the given bank
func (a *Account) WithdrawCredits(amount int) error {
	log.Trace("--> WithdrawCredits")
	defer log.Trace("<-- WithdrawCredits")

	a.mutex.Lock()
	defer a.mutex.Unlock()

	if a.CurrentBalance < amount {
		return ErrInsufficintBalance
	}
	a.MonthlyBalance -= amount
	a.CurrentBalance -= amount
	a.LifetimeBalance -= amount

	return nil
}

// LoadBanks returns all the banks for the given guilds.
func LoadBanks() {
	log.Trace("--> LoadBanks")
	defer log.Trace("<-- LoadBanks")

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
	log.Trace("--> SaveBank")
	defer log.Trace("<-- SaveBank")

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
