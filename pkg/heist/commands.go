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
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
)

// TODO: ensure heist commands are only run in the #heist channel
// TODO: check to see if Heist has been paused (it should be in the state)

var (
	servers map[string]*Server
	themes  map[string]*Theme
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
					Name:        "stats",
					Description: "Shows a user's stats",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "target",
					Description: "Commands that affect heist targets",
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "list",
							Description: "Gets the list of available heist targets",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "add",
							Description: "Adds a new target to the list of heist targets",
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
							Type: discordgo.ApplicationCommandOptionSubCommand,
						},
					},
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
				},
				{
					Name:        "theme",
					Description: "Commands that interact with the heist themes",
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
					Type: discordgo.ApplicationCommandOptionSubCommandGroup,
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
	case "clear":
		clearMember(s, i)
	case "reset":
		resetHeist(s, i)
	case "stats":
		playerStats(s, i)
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

/******** MESSAGE UTILITIES ********/

// commandFailure is a utility routine used to send an error response to a user's reaction to a bot's message.
func commandFailure(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	log.Debug("--> commandFailure")
	defer log.Debug("<-- commandFailure")

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
		timestamp := fmt.Sprintf("<t:%v:R> ", time.Now().Add(server.Config.WaitTime).Unix())
		status = "Starts " + timestamp
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

	var err error
	if action == "plan" {
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     embeds,
				Components: components,
			},
		})
	} else {
		s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds,
			Components: &components,
		})
	}
	if err != nil {
		return err
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
	if server.Heist != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "A " + theme.Heist + " is already being planned.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	server.Heist = NewHeist(player)
	server.Heist.Interaction = i
	server.Heist.Planned = true

	err := heistMessage(s, i, "plan")
	if err != nil {
		log.Error("Unable to create the `Plan Heist` message, error:", err)
	}

	server.Heist.Timer = newWaitTimer(s, i, server.Config.WaitTime, startHeist)

	store.SaveHeistState(server)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> joinHeist")
	defer log.Debug("<-- joinHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		commandFailure(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	if contains(server.Heist.Crew, player.ID) {
		commandFailure(s, i, "You are already a member of the "+theme.Heist+".")
		return
	}
	var err error

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
		commandFailure(s, i, "No "+theme.Heist+" is planned.")
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if server.Heist.Planner == player.ID {
		commandFailure(s, i, "You can't leave the "+theme.Heist+", as you are the planner.")
		return
	}
	if !contains(server.Heist.Crew, player.ID) {
		commandFailure(s, i, "You aren't a member of the "+theme.Heist+".")
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
		commandFailure(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	if i.Member.User.ID != server.Heist.Planner {
		log.Error("Unable to cancel heist, i.Member.User.ID:", i.Member.User.ID, ", server.Heist.Planner:", server.Heist.Planner)
		commandFailure(s, i, "You cannot cancel the "+theme.Heist+" as you are not the planner.")
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
	err := heistMessage(s, i, "start")
	if err != nil {
		log.Error("Unable to mark the heist message as started, error:", err)
	}

	// TODO: start the game.

	// For now, just clear out the heist so we can continue....
	time.Sleep(5 * time.Second)
	err = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
	if err != nil {
		log.Error("Unable to delete the heist message, error:", err)
	}
	server.Heist = nil

	store.SaveHeistState(server)
}

// playerStats shows a player's heist stats
func playerStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> playerStats")
	defer log.Debug("<-- playerStats")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
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
					Value:  strconv.Itoa(player.Spree),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Bail),
					Value:  strconv.Itoa(player.BailCost),
					Inline: true,
				},
				{
					Name:   caser.String(theme.OOB),
					Value:  strconv.FormatBool(player.OOB),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Sentence),
					Value:  strconv.Itoa(player.Sentence),
					Inline: true,
				},
				{
					Name:   "Apprehended",
					Value:  strconv.Itoa(player.JailCounter),
					Inline: true,
				},
				{
					Name:   "Death Timer",
					Value:  strconv.Itoa(player.DeathTimer),
					Inline: true,
				},
				{
					Name:   "Total Deaths",
					Value:  strconv.Itoa(player.Deaths),
					Inline: true,
				},
				{
					Name:   "Lifetime Apprehensions",
					Value:  strconv.Itoa(player.TotalJail),
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

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> resetHeist")
	defer log.Debug("<-- resetHeist")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
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
		return
	}

	server := GetServer(servers, i.GuildID)

	var id string
	var crewSize, vaultMax, vaultCurrent int
	var success float64
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		if option.Name == "id" {
			id = strings.TrimSpace(option.StringValue())
		} else if option.Name == "crew" {
			crewSize = int(option.IntValue())
		} else if option.Name == "success" {
			success = option.FloatValue()
		} else if option.Name == "vault" {
			vaultMax = int(option.IntValue())
		} else if option.Name == "current" {
			vaultCurrent = int(option.IntValue())
		}
	}
	if vaultCurrent == 0 {
		vaultCurrent = vaultMax
	}

	_, ok := server.Targets[id]
	if ok {
		commandFailure(s, i, "Target "+id+" already exists.")
		return
	}
	for _, target := range server.Targets {
		if target.CrewSize == crewSize {
			commandFailure(s, i, "Target "+target.ID+" has the same max crew size.")
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

// listTargets displays a list of available heist targets.
func listTargets(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listTargets")
	defer log.Debug("<-- listTargets")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		return
	}

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	if len(server.Targets) == 0 {
		msg := "There aren't any targets! To create a target use `/heist target add`."
		commandFailure(s, i, msg)
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

		data := []string{target.ID, strconv.Itoa(target.CrewSize), strconv.Itoa(target.Vault), strconv.Itoa(target.VaultMax), fmt.Sprintf("%.2f", target.Success)}
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
		return
	}

	var memberID string
	options := i.ApplicationCommandData().Options
	for _, option := range options {
		if option.Name == "playerID" {
			memberID = strings.TrimSpace(option.StringValue())
		}
	}
	server := GetServer(servers, i.GuildID)
	player, ok := server.Players[memberID]
	if !ok {
		commandFailure(s, i, "Player not found.")
		return
	}
	player.ClearSettings()
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
		commandFailure(s, i, "Theme `"+themeName+"` is already being used.")
		return
	}
	theme, err := LoadTheme(themeName)
	if err != nil {
		r := []rune(err.Error())
		r[0] = unicode.ToUpper(r[0])
		str := string(r)
		commandFailure(s, i, str)
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
	bot.Session.ApplicationCommandBulkOverwrite(appID, guildID, nil)
	log.Debug("Add new commands")
	bot.Session.ApplicationCommandBulkOverwrite(appID, guildID, commands)
}
