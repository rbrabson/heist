package economy

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
	"github.com/rbrabson/heist/pkg/math"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type leaderboardAccount struct {
	name    string
	balance int
}

// formatAccounts formats the leaderboard to be sent to a Discord server
func formatAccounts(p *message.Printer, title string, accounts []*leaderboardAccount) []*discordgo.MessageEmbed {
	log.Trace("--> formatAccounts")
	defer log.Trace("<-- formatAccounts")

	var tableBuffer strings.Builder
	table := tablewriter.NewWriter(&tableBuffer)
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetHeaderLine(false)
	table.SetBorder(false)
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.SetHeaderColor(tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor},
		tablewriter.Colors{tablewriter.FgBlueColor})

	table.SetHeader([]string{"#", "Name", "Balance"})
	for i, account := range accounts {
		data := []string{strconv.Itoa(i + 1), account.name, p.Sprintf("%d", account.balance)}
		table.Append(data)
	}
	table.Render()
	embeds := []*discordgo.MessageEmbed{
		{
			Type:  discordgo.EmbedTypeRich,
			Title: title,
			Fields: []*discordgo.MessageEmbedField{
				{
					Value: p.Sprintf("```\n%s```\n", tableBuffer.String()),
				},
			},
		},
	}

	return embeds
}

// getAccounts gets the list of accounts at a given bank
func getAccounts(bank *Bank) []*Account {
	log.Trace("--> getAccounts")
	defer log.Trace("<-- getAccounts")

	accounts := make([]*Account, 0, len(bank.Accounts))
	for _, account := range bank.Accounts {
		accounts = append(accounts, account)
	}
	return accounts
}

// getSortedAccounts converts a map of elements into a sorted list of those same elements.
func getSortedAccounts(accounts []*Account, sortFunc func(i, j int) bool) []*Account {
	log.Trace("--> getSortedAccounts")
	defer log.Trace("<-- getSortedAccounts")

	sort.Slice(accounts, sortFunc)
	return accounts
}

// GetRanking returns the ranking for the given member for various economy leaderboards.
// The sortFunc passed in determines how the rankings are determined.
func GetRanking(accounts []*Account, memberID string, sortFunc func(i, j int) bool) int {
	log.Trace("--> GetRanking")
	defer log.Trace("<-- GetRanking")

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
	log.Trace("--> GetMonthlyRanking")
	defer log.Trace("<-- GetMonthlyRanking")

	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].MonthlyBalance > accounts[j].MonthlyBalance
	})
	return rank
}

// GetCurrentRanking returns the global ranking on the server for a given player.
func GetCurrentRanking(serverID string, memberID string) int {
	log.Trace("--> GetCurrentRanking")
	defer log.Trace("<-- GetCurrentRanking")

	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	return rank
}

// GetLifetimeRanking returns the global ranking on the server for a given player.
func GetLifetimeRanking(serverID string, memberID string) int {
	log.Trace("--> GetLifetimeRanking")
	defer log.Trace("<-- GetLifetimeRanking")

	bank := banks[serverID]
	accounts := getAccounts(bank)
	rank := GetRanking(accounts, memberID, func(i, j int) bool {
		return accounts[i].CurrentBalance > accounts[j].CurrentBalance
	})
	return rank
}

// GetMonthlyLeaderboard returns the top `limit` accounts for the server.
func GetMonthlyLeaderboard(serverID string, limit int) []*leaderboardAccount {
	log.Trace("--> GetMonthlyLeaderboard")
	defer log.Trace("<-- GetMonthlyLeaderboard")

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
	log.Trace("--> GetCurrentLeaderboard")
	defer log.Trace("<-- GetCurrentLeaderboard")

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
	log.Trace("--> GetLifetimeLeaderboard")
	defer log.Trace("<-- GetLifetimeLeaderboard")

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
	log.Trace("--> resetMonthlyLeaderboard")
	defer log.Trace("<-- resetMonthlyLeaderboard")

	var lastSeason time.Time
	for _, bank := range banks {
		if lastSeason.Before(bank.LastSeason) {
			lastSeason = bank.LastSeason
		}
		break
	}
	if lastSeason.Year() == 1 {
		lastSeason = time.Now()
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
			// Trace last season's leaderboard
			accounts := GetMonthlyLeaderboard(bank.ID, 10)
			for i, account := range accounts {
				log.WithFields(log.Fields{
					"Rank":    i + 1,
					"Server":  bank.ID,
					"Account": account.name,
					"Balance": account.balance}).Info("Monthly Leaderboard Reset")
			}

			if bank.ChannelID != "" {
				p := message.NewPrinter(language.English)
				embeds := formatAccounts(p, fmt.Sprintf("%s %d Top 10", bank.LastSeason.Month().String(), bank.LastSeason.Year()), accounts)
				_, err := session.ChannelMessageSendComplex(bank.ChannelID, &discordgo.MessageSend{
					Embeds: embeds,
				})
				if err != nil {
					log.Error("Unable to send montly leaderboard, err:", err)
				}
			} else {
				log.WithField("guildID", bank.ChannelID).Warning("No leaderboard channel set for server")
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
