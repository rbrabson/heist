package race

import (
	"sort"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/message"
)

type leaderboardAccount struct {
	name    string
	balance int
}

// getLeaderboard returns the race leaderboard for all players on a given server
func getLeaderboard(serverID string, limit int) []leaderboardAccount {
	log.Trace("--> getLeaderboard")
	defer log.Trace("<-- getLeaderboard")

	server := GetServer(serverID)
	players := server.Players
	lb := make([]leaderboardAccount, 0, len(players))
	for _, player := range players {
		balance := player.Results.Earnings + (player.Results.BetEarnings - (server.Config.BetAmount * player.Results.BetsPlaced))
		lbAccount := leaderboardAccount{
			name:    player.Name,
			balance: balance,
		}
		lb = append(lb, lbAccount)
	}

	sort.Slice(lb, func(i, j int) bool {
		return lb[i].balance > lb[j].balance
	})

	return lb[:limit]
}

// formatAccounts formats the leaderboard to be sent to a Discord server
func formatAccounts(p *message.Printer, title string, accounts []leaderboardAccount) []*discordgo.MessageEmbed {
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
