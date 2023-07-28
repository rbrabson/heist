/*
commands contains the list of commands and messages sent to Discord, or commands processed when received from Discord.
*/
package heist

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rbrabson/heist/pkg/checks"
	"github.com/rbrabson/heist/pkg/economy"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
)

var (
	servers map[string]*Server
	themes  map[string]*Theme
	banks   map[string]*economy.Bank
	store   Store
	appID   string
)

// componentHandlers are the buttons that appear on messages sent by this bot.
var (
	componentsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"cancel_heist": cancelHeist,
		"join_heist":   joinHeist,
		"leave_heist":  leaveHeist,
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"heist": heist,
	}
	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "heist",
			Description: "Commands for the Heist bot",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "plan",
					Description: "Plans a heist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "reset",
					Description: "Resets a heist",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "player",
					Description: "Commands that affect individual players",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "stats",
							Description: "Shows a user's stats",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "bail",
							Description: "Bail a player out of jail",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "ID of the player to bail. Defaults to you.",
									Required:    false,
								},
							},
						},
						{
							Name:        "revive",
							Description: "Resurect player from the dead",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "release",
							Description: "Releases player from jail",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "clear",
							Description: "Clears the criminal settings for the user",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "ID of the player to clear",
									Required:    true,
								},
							},
							Type: discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "payday",
							Description: "Deposits your daily check into your bank account",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				{
					Name:        "target",
					Description: "Commands that affect heist targets",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "list",
							Description: "Gets the list of available heist targets",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "add",
							Description: "Adds a new target to the list of heist targets",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "ID of the heist",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "crew",
									Description: "Maximum crew size for the heist",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "success",
									Description: "Percentage liklihood of success (0..100)",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "vault",
									Description: "Maximum size of the target's vault",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "current",
									Description: "Current size of the target's vault; defaults to `vault`",
									Required:    false,
								},
							},
						},
					},
				},
				{
					Name:        "theme",
					Description: "Commands that interact with the heist themes",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "list",
							Description: "Gets the list of available heist themes",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "set",
							Description: "Sets the current heist theme",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "name",
									Description: "Name of the theme to set",
									Required:    true,
								},
							},
							Type: discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				{
					Name:        "version",
					Description: "Returns the version of heist running on the server",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}
)

/******** COMMAND ROUTER ********/

// heist routes subcommands to the appropriate interaction handler
func heist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> heist")
	defer log.Debug("<-- heist")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "plan":
		planHeist(s, i)
	case "reset":
		resetHeist(s, i)
	case "player":
		options = options[0].Options
		switch options[0].Name {
		case "clear":
			clearMember(s, i)
		case "stats":
			playerStats(s, i)
		case "bail":
			bailoutPlayer(s, i)
		case "release":
			releasePlayer(s, i)
		case "revive":
			revivePlayer(s, i)
		case "payday":
			payday(s, i)
		}
	case "target":
		options = options[0].Options
		switch options[0].Name {
		case "add":
			addTarget(s, i)
		case "list":
			listTargets(s, i)
		}
	case "theme":
		options = options[0].Options
		switch options[0].Name {
		case "list":
			listThemes(s, i)
		case "set":
			setTheme(s, i)
		}
	case "version":
		version(s, i)
	}
}

/******** UTILITY FUNCTIONS ********/

// getAssignedRoles returns a list of discord roles assigned to the user
func getAssignedRoles(s *discordgo.Session, i *discordgo.InteractionCreate) discordgo.Roles {
	guild, err := s.Guild(i.GuildID)
	if err != nil {
		log.Error("Unable to retrieve the guild information from Discord, error:", err)
		return nil
	}

	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
	if err != nil {
		log.Error("Unable to retrieve the member information from Discord, error:", err)
		return nil
	}

	var roles discordgo.Roles
	for _, role := range guild.Roles {
		if contains(member.Roles, role.ID) {
			roles = append(roles, role)
		}
	}

	return roles
}

// fmtDuration returns duration formatted for inclusion in Discord messages.
func fmtDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h == 1 {
		if m <= 30 {
			return "1 hour"
		}
		return "2 hours"
	}
	if h > 1 {
		if m > 30 {
			h++
		}
		if h == 1 {
			return "1 hour"
		}
		return fmt.Sprintf("%d hours", h)
	}
	if m > 1 {
		if s > 30 {
			m++
		}
		if m == 1 {
			return "1 minute"
		}
		return fmt.Sprintf("%d minutes", m)
	}
	if s <= 1 {
		return "1 second"
	}
	return fmt.Sprintf("%d seconds", s)
}

/******** MESSAGE UTILITIES ********/

// sendEphemeralResponse is a utility routine used to send an ephemeral response to a user's message or button press.
func sendNonephemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	log.Debug("--> sendNonephemeralResponse")
	defer log.Debug("<-- sendNonephemeralResponse")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
		},
	})
	if err != nil {
		log.Error("Unable to send a response, error:", err)
	}
}

// sendEphemeralResponse is a utility routine used to send an ephemeral response to a user's message or button press.
func sendEphemeralResponse(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	log.Debug("--> sendEphemeralResponse")
	defer log.Debug("<-- sendEphemeralResponse")

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to send a response, error:", err)
	}
}

// heistMessage sends the main command used to plan, join and leave a heist. It also handles the case where
// the heist starts, disabling the buttons to join/leave/cancel the heist.
func heistMessage(s *discordgo.Session, i *discordgo.InteractionCreate, action string) error {
	log.Debug("--> heistMessage")
	defer log.Debug("<-- heistMessage")

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	var status string
	var buttonDisabled bool
	if action == "plan" || action == "join" || action == "leave" {
		until := time.Until(server.Heist.StartTime)
		status = "Starts in " + fmtDuration(until)
		buttonDisabled = false
	} else if action == "update" {
		until := time.Until(server.Heist.StartTime)
		status = "Starts in " + fmtDuration(until)
		buttonDisabled = false
	} else if action == "start" {
		status = "Started"
		buttonDisabled = true
	} else {
		status = "Canceled"
		buttonDisabled = true
	}

	theme := themes[server.Config.Theme]
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Heist",
			Description: "A new " + theme.Heist + " is being planned by " + player.Name + ". You can join the " + theme.Heist + " at any time prior to the " + theme.Heist + " starting.",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  status,
					Inline: true,
				},
				{
					Name:   "Number of " + caser.String(theme.Crew) + "  Members",
					Value:  strconv.Itoa(len(server.Heist.Crew)),
					Inline: true,
				},
			},
		},
	}
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.Button{
				Label:    "Join",
				Style:    discordgo.SuccessButton,
				Disabled: buttonDisabled,
				CustomID: "join_heist",
			},
			discordgo.Button{
				Label:    "Leave",
				Style:    discordgo.PrimaryButton,
				Disabled: buttonDisabled,
				CustomID: "leave_heist"},
			discordgo.Button{
				Label:    "Cancel",
				Style:    discordgo.DangerButton,
				Disabled: buttonDisabled,
				CustomID: "cancel_heist"},
		}},
	}

	if action == "plan" {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     embeds,
				Components: components,
			},
		})
		if err != nil {
			return err
		}
	} else {
		_, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds,
			Components: &components,
		})
		if err != nil {
			return err
		}
	}

	return nil
}

/******** PLAYER COMMANDS ********/

// planHeist plans a new heist
func planHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> planHeist")
	defer log.Debug("<-- planHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	// Heist is already in progress
	if server.Heist != nil {
		sendEphemeralResponse(s, i, "A "+theme.Heist+" is already being planned.")
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	// Basic error checks for the heist
	if msg, ok := heistChecks(server, player, server.Targets); !ok {
		sendEphemeralResponse(s, i, msg)
		return
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(banks, server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	err := economy.WithdrawCredits(bank, account, int(server.Config.HeistCost))
	if err != nil {
		log.Errorf("Unable to withdraw credits for the heist from the account of %s, error=%s", player.ID, err.Error())
	}

	server.Heist = NewHeist(server, player)
	server.Heist.Interaction = i
	server.Heist.Planned = true

	err = heistMessage(s, i, "plan")
	if err != nil {
		log.Error("Unable to create the `Plan Heist` message, error:", err)
	}

	server.Heist.Timer = newWaitTimer(s, i, time.Until(server.Heist.StartTime), startHeist)

	store.SaveHeistState(server)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> joinHeist")
	defer log.Debug("<-- joinHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		sendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	if contains(server.Heist.Crew, player.ID) {
		sendEphemeralResponse(s, i, "You are already a member of the "+theme.Heist+".")
		return
	}
	if msg, ok := heistChecks(server, player, server.Targets); !ok {
		sendEphemeralResponse(s, i, msg)
		return
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(banks, server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	err := economy.WithdrawCredits(bank, account, int(server.Config.HeistCost))
	if err != nil {
		log.Errorf("Unable to withdraw credits to join the heist from the account of %s, error=%s", player.ID, err.Error())
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have joined the " + theme.Heist + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to notify the member they have joined the heist, error:", err)
	}

	server.Heist.Crew = append(server.Heist.Crew, player.ID)
	err = heistMessage(s, server.Heist.Interaction, "join")
	if err != nil {
		log.Error("Unable to update the heist message, error:", err)
	}

	store.SaveHeistState(server)
}

// leaveHeist attempts to leave a heist previously joined
func leaveHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> leaveHeist")
	defer log.Debug("<-- leaveHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		log.Error("There should be a heist, server:", server.ID, ", heist:", server.Heist)
		sendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if server.Heist.Planner == player.ID {
		sendEphemeralResponse(s, i, "You can't leave the "+theme.Heist+", as you are the planner.")
		return
	}
	if !contains(server.Heist.Crew, player.ID) {
		sendEphemeralResponse(s, i, "You aren't a member of the "+theme.Heist+".")
		return
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have left the " + theme.Heist + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to notify the user they have left the heist, error:", err)
	}
	server.Heist.Crew = remove(server.Heist.Crew, player.ID)

	err = heistMessage(s, server.Heist.Interaction, "leave")
	if err != nil {
		log.Error("Unable to update the heist message, error:", err)
	}

	store.SaveHeistState(server)
}

// cancelHeist cancels a heist that is being planned but has not yet started
func cancelHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> cancelHeist")
	defer log.Debug("<-- cancelHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		sendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	if i.Member.User.ID != server.Heist.Planner {
		log.Error("Unable to cancel heist, i.Member.User.ID:", i.Member.User.ID, ", server.Heist.Planner:", server.Heist.Planner)
		sendEphemeralResponse(s, i, "You cannot cancel the "+theme.Heist+" as you are not the planner.")
		return
	}

	err := heistMessage(s, server.Heist.Interaction, "cancel")
	if err != nil {
		log.Error("Unable to mark the heist message as cancelled, error:", err)
	}
	server.Heist.Timer.cancel()
	server.Heist = nil

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "The " + theme.Heist + " has been cancelled.",
		},
	})
	if err != nil {
		log.Error("Unable to notify the user the heist has been cancelled, error:", err)
	}

	store.SaveHeistState(server)
}

// startHeist is called once the wait time for planning the heist completes
func startHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> startHeist")
	defer log.Debug("<-- startHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		s.ChannelMessageSend(i.ChannelID, "Error: no heist found.")
		heistMessage(s, i, "cancel")
		return
	}
	if len(server.Targets) == 1 {
		sendEphemeralResponse(s, i, "There are no heist targets. Add one using the `/heist target add` command.")
		server.Heist = nil
		return
	}

	server.Heist.Started = true
	server.Heist.Planned = false

	err := heistMessage(s, i, "start")
	if err != nil {
		log.Error("Unable to mark the heist message as started, error:", err)
	}

	var msg string
	if len(server.Heist.Crew) <= 1 {
		msg = fmt.Sprintf("You tried to rally a %s, but no one wanted to follow you. The %s has been cancelled.", theme.Crew, theme.Heist)
		s.ChannelMessageSend(i.ChannelID, msg)
		server.Heist = nil
		return
	}

	target := getTarget(server.Heist, server.Targets)
	results := getHeistResults(server, target)
	msg = fmt.Sprintf("Get ready! The %s is starting.\nThe %s has decided to hit **%s**.", theme.Heist, theme.Crew, target.ID)
	s.ChannelMessageSend(i.ChannelID, msg)

	time.Sleep(3 * time.Second)

	// Process the results
	for _, result := range results.memberResults {
		msg = fmt.Sprintf(result.message+"\n", result.player.Name)
		s.ChannelMessageSend(i.ChannelID, msg)
		time.Sleep(5 * time.Second)
	}

	if len(results.survivingCrew) == 0 {
		msg = "No one made it out safe."
		s.ChannelMessageSend(i.ChannelID, msg)
	} else {
		embeds := make([]*discordgo.MessageEmbed, 0, len(results.survivingCrew)+1)
		for _, result := range results.survivingCrew {
			embed := discordgo.MessageEmbed{
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Player",
						Value:  result.player.Name,
						Inline: true,
					},
					{
						Name:   "Credits",
						Value:  strconv.Itoa(result.stolenCredits + result.bonusCredits),
						Inline: true,
					},
				},
			}
			embeds = append(embeds, &embed)
			log.WithFields(log.Fields{
				"Player":           result.player.Name,
				"Credits Obtained": result.stolenCredits,
				"Bonus":            result.bonusCredits,
				"Total":            result.stolenCredits + result.bonusCredits,
			}).Debug("Result")
		}
		data := &discordgo.MessageSend{
			Content: "**Heist Payout**",
			Embeds:  embeds,
		}
		_, err = s.ChannelMessageSendComplex(i.ChannelID, data)
		if err != nil {
			log.Error("Failed to send heist resuilts, error:", err)
		}
	}

	// set server.Config.AlertTime to the current time

	// Table like the following:
	// Player       Credits Obtained        Bonuses       Total
	// Table title will be "The credits collected from the %s was split among the winners", theme.Heist

	// get the bank from the economy and deposit credits into the surviving crew's money
	// subtract off the amount stolen from the target
	// update the player's status based on the results

	// Reset the heist
	server.Heist = nil
	store.SaveHeistState(server)

	// Unmute the channel (only if I mute it to start with)

}

// playerStats shows a player's heist stats
func playerStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> playerStats")
	defer log.Debug("<-- playerStats")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))

	bank := economy.GetBank(banks, server.ID)
	account := bank.GetAccount(player.ID, player.Name)

	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Player Stats",
			Description: "Current stats for " + player.Name + ".",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  player.Status,
					Inline: true,
				},
				{
					Name:   "Spree",
					Value:  strconv.Itoa(int(player.Spree)),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Bail),
					Value:  strconv.Itoa(int(player.BailCost)),
					Inline: true,
				},
				{
					Name:   caser.String(theme.OOB),
					Value:  strconv.FormatBool(player.OOB),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Sentence),
					Value:  strconv.Itoa(int(player.Sentence)),
					Inline: true,
				},
				{
					Name:   "Apprehended",
					Value:  strconv.Itoa(int(player.JailCounter)),
					Inline: true,
				},
				{
					Name:   "Total Deaths",
					Value:  strconv.Itoa(int(player.Deaths)),
					Inline: true,
				},
				{
					Name:   "Lifetime Apprehensions",
					Value:  strconv.Itoa(int(player.TotalJail)),
					Inline: true,
				},
				{
					Name:   "Credits",
					Value:  strconv.Itoa(account.Balance),
					Inline: true,
				},
			},
		},
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to send the player stats to Discord, error:", err)
	}
}

// bailoutPlayer bails a player player out from jail. This defaults to the player initiating the command, but can
// be another player as well.
func bailoutPlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> bailoutPlayer")
	log.Debug("<-- bailoutPlayer")

	var playerID string
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		if option.Name == "id" {
			playerID = strings.TrimSpace(option.StringValue())
		}
	}

	server := GetServer(servers, i.GuildID)
	initiatingPlayer := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	bank := economy.GetBank(banks, server.ID)
	account := bank.GetAccount(initiatingPlayer.ID, initiatingPlayer.Name)

	var player *Player
	if playerID != "" {
		var ok bool
		player, ok = server.Players[playerID]
		if !ok {
			sendEphemeralResponse(s, i, "Player "+playerID+" does not exist.")
			return
		}
	} else {
		player = initiatingPlayer
	}

	if player.Status != "Apprehended" || player.OOB {
		var msg string
		if player.ID == i.Member.User.ID {
			msg = "You are not in jail"
		} else {
			msg = fmt.Sprintf("%s is not in jail", player.Name)
		}
		sendEphemeralResponse(s, i, msg)
		return
	}

	if account.Balance < int(player.BailCost) {
		msg := fmt.Sprintf("You do not have enough credits to play the bail of %d", player.BailCost)
		sendEphemeralResponse(s, i, msg)
		return
	}

	economy.WithdrawCredits(bank, account, int(player.BailCost))
	player.OOB = true

	msg := fmt.Sprintf("Congratulations, %s, %s bailed you out and now you are free!. Enjoy your freedom while it lasts", player.Name, initiatingPlayer.Name)
	sendNonephemeralResponse(s, i, msg)
}

// releasePlayer releases a player from jail if their sentence has been served.
func releasePlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> releasePlayer")
	defer log.Debug("<-- releasePlayer")

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	theme := themes[server.Config.Theme]

	if player.Status != "Apprehended" || player.OOB {
		sendEphemeralResponse(s, i, "I can't remove you from jail if you're not *in* jail")
		return
	}
	if player.JailTimer.After(time.Now()) {
		remainingTime := time.Until(player.JailTimer)
		msg := fmt.Sprintf("You still have time on your %s, you still need to wait %s.", theme.Sentence, fmtDuration(remainingTime))
		sendEphemeralResponse(s, i, msg)
		return
	}
	msg := "You served your time. Enjoy the fresh air of freedom while you can."
	if player.OOB {
		msg += "/nYou are no longer on probabtion! 3x penalty removed."
	}
	sendEphemeralResponse(s, i, msg)
}

// revivePlayer raises a player from the dead if their death timer has expired.
func revivePlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> revivePlayer")
	defer log.Debug("<-- revivePlayer")

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if player.Status != "Dead" {
		sendEphemeralResponse(s, i, "You still have a pulse. I can't reive someone who isn't dead.")
		return
	}
	if player.DeathTimer.After(time.Now()) {
		remainingTime := time.Until(player.DeathTimer)
		msg := fmt.Sprintf("You can't revive yet. You need to wait %s", fmtDuration(remainingTime))
		sendEphemeralResponse(s, i, msg)
		return
	}
	player.ClearJailAndDeathStatus()
	sendEphemeralResponse(s, i, "You have risen from the dead!")
}

// payday gives some credits to the player every 24 hours.
func payday(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> payday")
	defer log.Debug("<-- payday")

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if player.PaydayTimer.After(time.Now()) {
		remainingTime := time.Until(player.PaydayTimer)
		msg := fmt.Sprintf("You can't get another payday yet. You need to wait %s.", fmtDuration(remainingTime))
		sendEphemeralResponse(s, i, msg)
		return
	}

	bank := economy.GetBank(banks, server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	economy.DepositCredits(bank, account, server.Config.PaydayAmount)
	player.PaydayTimer = time.Now().Add(24 * time.Hour)
	store.SaveHeistState(server)

	msg := fmt.Sprintf("You deposited your check of %d into your bank account. You now have %d credits.", server.Config.PaydayAmount, account.Balance)
	sendEphemeralResponse(s, i, msg)
}

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> resetHeist")
	defer log.Debug("<-- resetHeist")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}
	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil || !server.Heist.Planned {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No " + theme.Heist + " is being planned.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Error("Unable to notify the user no heist is being planned, error:", err)
		}
		return
	}

	if server.Heist.Timer != nil {
		server.Heist.Timer.cancel()
	}
	heistMessage(s, server.Heist.Interaction, "cancel")
	server.Heist = nil

	if server.Heist == nil || !server.Heist.Planned {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "The " + theme.Heist + " has been reset.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Error("Unable to notify the user the heist has been resset, error:", err)
		}
	}

	store.SaveHeistState(server)
}

// addTarget adds a target for heists
func addTarget(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> addTarget")
	defer log.Debug("<-- addTarget")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	server := GetServer(servers, i.GuildID)

	var id string
	var crewSize, vaultMax, vaultCurrent int64
	var success float64
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		if option.Name == "id" {
			id = strings.TrimSpace(option.StringValue())
		} else if option.Name == "crew" {
			crewSize = option.IntValue()
		} else if option.Name == "success" {
			success = option.FloatValue()
		} else if option.Name == "vault" {
			vaultMax = option.IntValue()
		} else if option.Name == "current" {
			vaultCurrent = option.IntValue()
		}
	}
	if vaultCurrent == 0 {
		vaultCurrent = vaultMax
	}

	_, ok := server.Targets[id]
	if ok {
		sendEphemeralResponse(s, i, "Target "+id+" already exists.")
		return
	}
	for _, target := range server.Targets {
		if target.CrewSize == crewSize {
			sendEphemeralResponse(s, i, "Target "+target.ID+" has the same max crew size.")
			return
		}

	}

	target := NewTarget(id, crewSize, success, vaultCurrent, vaultMax)
	server.Targets[target.ID] = target

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have added target " + target.ID + " to the new heist.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to notify the user the new target has been added, error:", err)
	}

	store.SaveHeistState(server)
}

// TODO: editTarget
// TODO: removeTarget

// listTargets displays a list of available heist targets.
func listTargets(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listTargets")
	defer log.Debug("<-- listTargets")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	if len(server.Targets) == 0 {
		msg := "There aren't any targets! To create a target use `/heist target add`."
		sendEphemeralResponse(s, i, msg)
		return
	}

	targets := make([]*Target, 0, len(server.Targets))
	for _, target := range server.Targets {
		targets = append(targets, target)
	}
	sort.SliceStable(targets, func(i, j int) bool {
		return targets[i].CrewSize < targets[j].CrewSize
	})

	// Lets return the data in an Ascii table. Ideally, it would be using a Discord embed, but unfortunately
	// Discord only puts three columns per row, which isn't enough for our purposes.
	var tableBuffer strings.Builder
	table := tablewriter.NewWriter(&tableBuffer)
	table.SetHeader([]string{"ID", "Max Crew", theme.Vault, "Max " + theme.Vault, "Success Rate"})
	for _, target := range targets {

		data := []string{target.ID, strconv.Itoa(int(target.CrewSize)), strconv.Itoa(int(target.Vault)), strconv.Itoa(int(target.VaultMax)), fmt.Sprintf("%.2f", target.Success)}
		table.Append(data)
	}
	table.Render()

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "```\n" + tableBuffer.String() + "\n```",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to sent the list of targets, error:", err)
	}
}

// clearMember clears the criminal state of the player.
func clearMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> clearMember")
	log.Debug("<-- clearMember")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	var memberID string
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		if option.Name == "id" {
			memberID = strings.TrimSpace(option.StringValue())
		}
	}
	server := GetServer(servers, i.GuildID)
	player, ok := server.Players[memberID]
	if !ok {
		sendEphemeralResponse(s, i, "Player not found.")
		return
	}
	player.Reset()
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Player settings cleared.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to send message that the player settings have been cleared, error:", err)
	}

	store.SaveHeistState(server)
}

// listThemes returns the list of available themes that may be used for heists
func listThemes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listThemes")
	defer log.Debug("<-- listThemes")
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	themes, err := GetThemeNames(themes)
	if err != nil {
		log.Warning("Unable to get the themes, error:", err)
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Available Themes",
			Description: "Available Themes for the Heist bot",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Themes",
					Value:  strings.Join(themes[:], ", "),
					Inline: true,
				},
			},
		},
	}

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to send list of themes to the user, error:", err)
	}
}

// setTheme sets the heist theme to the one specified in the command
func setTheme(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> setTheme")
	defer log.Debug("<-- setTheme")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	server := GetServer(servers, i.GuildID)
	var themeName string
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		if option.Name == "name" {
			themeName = strings.TrimSpace(option.StringValue())
		}
	}

	if themeName == server.Config.Theme {
		sendEphemeralResponse(s, i, "Theme `"+themeName+"` is already being used.")
		return
	}
	theme, err := LoadTheme(themeName)
	if err != nil {
		r := []rune(err.Error())
		r[0] = unicode.ToUpper(r[0])
		str := string(r)
		sendEphemeralResponse(s, i, str)
		return
	}
	server.Config.Theme = theme.ID
	log.Info("Now using theme", server.Config.Theme)

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Theme " + themeName + " is now being used.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to notify user that the selected theme is now being used, error:", err)
	}

	store.SaveHeistState(server)
}

// version shows the version of heist you are running.
func version(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> version")
	defer log.Debug("<-- version")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		sendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}
	server := GetServer(servers, i.GuildID)

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You are running Heist version " + server.Config.Version + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error("Unable to send the Heist version to the user, error:", err)
	}
}

// addBotCommands adds all commands that may be issued from a given server.
func addBotCommands(bot *Bot) {
	log.Debug("adding bot commands")

	appID = os.Getenv("APP_ID")
	store = NewStore()
	servers = LoadServers(store)
	themes = LoadThemes(store)
	banks = economy.LoadBanks()

	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Heist bot is up!")
	})
	bot.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			if h, ok := componentsHandlers[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}
		}
	})

	// Delete any old slash commands, and then add in my current set
	guildID := os.Getenv("HEIST_GUILD_ID")
	log.Debug("Delete old commands")
	_, err := bot.Session.ApplicationCommandBulkOverwrite(appID, guildID, nil)
	if err != nil {
		log.Fatal("Failed to delete all old commands, error:", err)
	}
	log.Debug("Add new commands")
	_, err = bot.Session.ApplicationCommandBulkOverwrite(appID, guildID, commands)
	if err != nil {
		log.Fatal("Failed to load new commands, error:", err)
	}
}
