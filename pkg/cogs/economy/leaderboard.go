package economy

import (
	"sort"
	"time"

	"github.com/rbrabson/heist/pkg/math"
	log "github.com/sirupsen/logrus"
)

type leaderboardAccount struct {
	name    string
	balance int
}

// getAccounts gets the list of accounts at a given bank
func getAccounts(bank *Bank) []*Account {
	accounts := make([]*Account, 0, len(bank.Accounts))
	for _, account := range bank.Accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// getSortedAccounts converts a map of elements into a sorted list of those same elements.
func getSortedAccounts(accounts []*Account, sortFunc func(i, j int) bool) []*Account {
	sort.Slice(accounts, sortFunc)
	return accounts
}

// GetRanking returns the ranking for the given member for various economy leaderboards.
// The sortFunc passed in determines how the rankings are determined.
func GetRanking(accounts []*Account, memberID string, sortFunc func(i, j int) bool) int {
	getSortedAccounts(accounts, sortFunc)
	var rank int
	for i := range accounts {
		rank = i + 1
		if accounts[i].ID == memberID {
			break
		}
	}
	return rank
}

// GetMonthlyRanking returns the global ranking on the server for a given player.
func GetMonthlyRanking(serverID string, memberID string) int {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].MonthlyBalance > accounts[j].MonthlyBalance
	})
	return rank
}

// GetCurrentRanking returns the global ranking on the server for a given player.
func GetCurrentRanking(serverID string, memberID string) int {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	return rank
}

// GetLifetimeRanking returns the global ranking on the server for a given player.
func GetLifetimeRanking(serverID string, memberID string) int {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	return rank
}

// GetMonthlyLeaderboard returns the top `limit` accounts for the server.
func GetMonthlyLeaderboard(serverID string, limit int) []*leaderboardAccount {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].MonthlyBalance > accounts[j].MonthlyBalance
	})
	num := math.Min(limit, len(accounts))
	leaderboard := make([]*leaderboardAccount, 0, num)
	for _, account := range accounts[:num] {
		a := leaderboardAccount{
			name:    account.Name,
			balance: account.MonthlyBalance,
		}
		leaderboard = append(leaderboard, &a)
	}
	return leaderboard
}

// GetCurrentLeaderboard returns the top `limit` accounts for the server.
func GetCurrentLeaderboard(serverID string, limit int) []*leaderboardAccount {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	num := math.Min(limit, len(accounts))
	leaderboard := make([]*leaderboardAccount, 0, num)
	for _, account := range accounts[:num] {
		a := leaderboardAccount{
			name:    account.Name,
			balance: account.CurrentBalance,
		}
		leaderboard = append(leaderboard, &a)
	}
	return leaderboard
}

// GetLifetimeLeaderboard returns the top `limit` accounts for the server.
func GetLifetimeLeaderboard(serverID string, limit int) []*leaderboardAccount {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].LifetimeBalance > accounts[j].LifetimeBalance
	})
	num := math.Min(limit, len(accounts))
	leaderboard := make([]*leaderboardAccount, 0, num)
	for _, account := range accounts[:num] {
		a := leaderboardAccount{
			name:    account.Name,
			balance: account.LifetimeBalance,
		}
		leaderboard = append(leaderboard, &a)
	}
	return leaderboard
}

// resetMonthlyLeaderboard resets the MonthlyBalance for all accounts to zero.
func resetMonthlyLeaderboard() {
	// TODO: need some work here. I need to handle restarts, for certain. What happens if
	// the bot is down when the new month starts? Gotta handle that, I think. Or maybe not.
	// Look through the logic on that edge case.
	var lastSeason time.Time
	for _, bank := range banks {
		if lastSeason.Before(bank.LastSeason) {
			lastSeason = bank.LastSeason
		}
		break
	}
	month := lastSeason.Month()
	year := lastSeason.Year()

	for {
		month++
		if month > time.December {
			month = time.January
			year++
		}
		log.WithFields(log.Fields{"Month": month, "Year": year}).Debug("Reset Economy On Date")

		nextMonth := time.Date(year, month, 1, 0, 0, 0, 0, time.UTC)
		time.Sleep(time.Until(nextMonth))

		for _, bank := range banks {
			// Trace last season's leaderboard. Right now, I don't have a good way to output this to a server's channel
			for i, account := range GetMonthlyLeaderboard(bank.ID, 10) {
				log.WithFields(log.Fields{"Rank": i + 1, "Server": bank.ID, "Account": account.name, "Balance": account.balance}).Info("Monthly Leaderboard Reset")
			}

			bank.LastSeason = nextMonth
			for _, account := range bank.Accounts {
				account.mutex.Lock()
				account.MonthlyBalance = 0
				account.mutex.Unlock()
			}
			SaveBank(bank)
		}
	}
}
