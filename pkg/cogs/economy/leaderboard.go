package economy

import (
	"sort"
	"time"
)

type number interface {
	int | int32 | int64 | float32 | float64 | time.Duration
}

func min[N number](v1 N, v2 N) N {
	if v1 < v2 {
		return v1
	}
	return v2
}

// convertToList converts a map of elements into a sorted list of those same elements.
func convertToList(serverID string) []*Account {
	bank := banks[serverID]
	accounts := make([]*Account, 0, len(bank.Accounts))
	for _, account := range bank.Accounts {
		accounts = append(accounts, account)
	}
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].Balance > accounts[j].Balance
	})
	return accounts
}

// GetRanking returns the global ranking on the server for a given player.
func GetRanking(serverID string, memberID string) int {
	var rank int
	accounts := convertToList(serverID)
	for i := range accounts {
		rank = i + 1
		if accounts[i].ID == memberID {
			break
		}
	}
	return rank
}

// GetLeaderboard returns the top `limit` accounts for the server.
func GetLeaderboard(serverID string, limit int) []*Account {
	accounts := convertToList(serverID)
	num := min(limit, len(accounts))
	return accounts[:num]
}
