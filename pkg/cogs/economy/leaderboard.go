package economy

import (
	"sort"

	"github.com/rbrabson/heist/pkg/math"
)

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
func GetMonthlyLeaderboard(serverID string, limit int) []*Account {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].MonthlyBalance > accounts[j].MonthlyBalance
	})
	num := math.Min(limit, len(accounts))
	return accounts[:num]
}

// GetCurrentLeaderboard returns the top `limit` accounts for the server.
func GetCurrentLeaderboard(serverID string, limit int) []*Account {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	num := math.Min(limit, len(accounts))
	return accounts[:num]
}

// GetLifetimeLeaderboard returns the top `limit` accounts for the server.
func GetLifetimeLeaderboard(serverID string, limit int) []*Account {
	bank := banks[serverID]
	accounts := getAccounts(bank)
	getSortedAccounts(accounts, func(i, j int) bool {
		return accounts[i].LifetimeBalance > accounts[j].LifetimeBalance
	})
	num := math.Min(limit, len(accounts))
	return accounts[:num]
}
