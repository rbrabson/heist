/*
commands contains the list of commands and messages sent to Discord, or commands processed when received from Discord.
*/
package heist

import (
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"golang.org/x/text/message"

	"github.com/rbrabson/heist/pkg/checks"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/cogs/payday"
	"github.com/rbrabson/heist/pkg/format"
	discmsg "github.com/rbrabson/heist/pkg/msg"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
	"github.com/olekukonko/tablewriter"
)

const (
	HEIST = "heist"
)

var (
	servers map[string]*Server
	themes  map[string]*Theme
)

// componentHandlers are the buttons that appear on messages sent by this bot.
var (
	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"cancel_heist": cancelHeist,
		"join_heist":   joinHeist,
		"leave_heist":  leaveHeist,
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"heist":       heist,
		"heist-admin": admin,
	}

	playerCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "heist",
			Description: "Heist game commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "bail",
					Description: "Bail a player out of jail.",
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
					Description: "Resurrect player from the dead.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "release",
					Description: "Releases player from jail.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stats",
					Description: "Shows a user's stats.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "start",
					Description: "Plans a new heist.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	adminCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "heist-admin",
			Description: "Heist admin commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "clear",
					Description: "Clears the criminal settings for the user.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Type:        discordgo.ApplicationCommandOptionString,
							Name:        "id",
							Description: "ID of the player to clear",
							Required:    true,
						},
					},
				},
				{
					Name:        "config",
					Description: "Configures the Heist bot.",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "info",
							Description: "Returns the configuration information for the server.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "bail",
							Description: "Sets the base cost of bail.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "amount",
									Description: "The base cost of bail.",
									Required:    true,
								},
							},
						},
						{
							Name:        "cost",
							Description: "Sets the cost to plan or join a heist.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "amount",
									Description: "The cost to plan or join a heist.",
									Required:    true,
								},
							},
						},
						{
							Name:        "death",
							Description: "Sets how long players remain dead.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "time",
									Description: "The time the player remains dead, in seconds.",
									Required:    true,
								},
							},
						},
						{
							Name:        "patrol",
							Description: "Sets the time the authorities will prevent a new heist.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "time",
									Description: "The time the authorities will patrol, in seconds.",
									Required:    true,
								},
							},
						},
						{
							Name:        "payday",
							Description: "Sets how many credits a player gets for each payday.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "amount",
									Description: "The amount deposited in a players account for each payday.",
									Required:    true,
								},
							},
						},
						{
							Name:        "sentence",
							Description: "Sets the base apprehension time when caught.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "time",
									Description: "The base time, in seconds.",
									Required:    true,
								},
							},
						},
						{
							Name:        "wait",
							Description: "Sets how long players can gather others for a heist.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "time",
									Description: "The time to wait for players to join the heist, in seconds.",
									Required:    true,
								},
							},
						},
					},
				},
				{
					Name:        "target",
					Description: "Commands that affect heist targets.",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "list",
							Description: "Gets the list of available heist targets.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "add",
							Description: "Adds a new target to the list of heist targets.",
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
									Description: "Maximum crew size for the heist.",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "success",
									Description: "Percentage liklihood of success (0..100).",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "vault",
									Description: "Maximum size of the target's vault.",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "current",
									Description: "Current size of the target's vault; defaults to `vault`.",
									Required:    false,
								},
							},
						},
						{
							Name:        "edit",
							Description: "Edits an existing heist target.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "ID of the heist.",
									Required:    true,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "crew",
									Description: "Maximum crew size for the heist.",
									Required:    false,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "success",
									Description: "Percentage liklihood of success (0..100).",
									Required:    false,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "vault",
									Description: "Maximum size of the target's vault.",
									Required:    false,
								},
								{
									Type:        discordgo.ApplicationCommandOptionInteger,
									Name:        "current",
									Description: "Current size of the target's vault; defaults to `vault`.",
									Required:    false,
								},
							},
						},
						{
							Name:        "remove",
							Description: "Removes a target from the list of heist targets.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "ID of the heist.",
									Required:    true,
								},
							},
						},
					},
				},
				{
					Name:        "theme",
					Description: "Commands that interact with the heist themes.",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "list",
							Description: "Gets the list of available heist themes.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "set",
							Description: "Sets the current heist theme.",
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "name",
									Description: "Name of the theme to set.",
									Required:    true,
								},
							},
							Type: discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				{
					Name:        "reset",
					Description: "Resets a new heist that is hung.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}
)

/******** COMMAND ROUTERS ********/

// config routes the configuration commands to the proper handlers.
func config(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> config")
	defer log.Debug("<-- config")

	options := i.ApplicationCommandData().Options[0].Options
	switch options[0].Name {
	case "cost":
		configCost(s, i)
	case "sentence":
		configSentence(s, i)
	case "patrol":
		configPatrol(s, i)
	case "bail":
		configBail(s, i)
	case "death":
		configDeath(s, i)
	case "wait":
		configWait(s, i)
	case "payday":
		configPayday(s, i)
	case "info":
		configInfo(s, i)
	}
}

// target routes the target commands to the proper handlers.
func target(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> target")
	defer log.Debug("<-- target")

	options := i.ApplicationCommandData().Options[0].Options
	switch options[0].Name {
	case "add":
		addTarget(s, i)
	case "edit":
		editTarget(s, i)
	case "remove":
		removeTarget(s, i)
	case "list":
		listTargets(s, i)
	}
}

// theme routes the theme commands to the proper handlers.
func theme(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> theme")
	defer log.Debug("<-- theme")

	options := i.ApplicationCommandData().Options[0].Options
	switch options[0].Name {
	case "list":
		listThemes(s, i)
	case "set":
		setTheme(s, i)
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

// getPrinter returns a printer for the given locale of the user initiating the message.
func getPrinter(i *discordgo.InteractionCreate) *message.Printer {
	tag, err := language.Parse(string(i.Locale))
	if err != nil {
		log.Error("Unable to parse locale, error:", err)
		tag = language.English
	}
	return message.NewPrinter(tag)
}

/******** MESSAGE UTILITIES ********/

// heistMessage sends the main command used to plan, join and leave a heist. It also handles the case where
// the heist starts, disabling the buttons to join/leave/cancel the heist.
func heistMessage(s *discordgo.Session, i *discordgo.InteractionCreate, action string) error {
	log.Debug("--> heistMessage")
	defer log.Debug("<-- heistMessage")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	var status string
	var buttonDisabled bool
	if action == "plan" || action == "join" || action == "leave" {
		until := time.Until(server.Heist.StartTime)
		status = "Starts in " + format.Duration(until)
		buttonDisabled = false
	} else if action == "update" {
		until := time.Until(server.Heist.StartTime)
		status = "Starts in " + format.Duration(until)
		buttonDisabled = false
	} else if action == "start" {
		status = "Started"
		buttonDisabled = true
	} else if action == "cancel" {
		status = "Canceled"
		buttonDisabled = true
	} else {
		status = "Ended"
		buttonDisabled = true
	}

	theme := themes[server.Config.Theme]
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	msg := p.Sprintf("A new %s is being planned by %s. You can join the %s for a cost of %d credits at any time to the %s startting.", theme.Heist, player.Name, theme.Heist, server.Config.HeistCost, theme.Heist)
	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Heist",
			Description: msg,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  status,
					Inline: true,
				},
				{
					Name:   "Number of " + caser.String(theme.Crew) + "  Members",
					Value:  p.Sprintf("%d", len(server.Heist.Crew)),
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

/******** COMMAND ROUTERS ********/

// admin routes the commands to the subcommand and subcommandgroup handlers
func admin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> admin")
	defer log.Debug("<-- admin")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "clear":
		clearMember(s, i)
	case "config":
		config(s, i)
	case "reset":
		resetHeist(s, i)
	case "target":
		target(s, i)
	case "theme":
		theme(s, i)
	}
}

// heist routes the commands to the subcommand and subcommandgroup handlers
func heist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> heist")
	defer log.Debug("<-- heist")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "bail":
		bailoutPlayer(s, i)
	case "release":
		releasePlayer(s, i)
	case "revive":
		revivePlayer(s, i)
	case "start":
		planHeist(s, i)
	case "stats":
		playerStats(s, i)
	}
}

/******** PLAYER COMMANDS ********/

// planHeist plans a new heist“
func planHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> planHeist")
	defer log.Debug("<-- planHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	// Heist is already in progress
	if server.Heist != nil {
		discmsg.SendEphemeralResponse(s, i, "A "+theme.Heist+" is already being planned.")
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	// Basic error checks for the heist
	if msg, ok := heistChecks(server, i, player, server.Targets); !ok {
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	economy.WithdrawCredits(bank, account, int(server.Config.HeistCost))
	economy.SaveBank(bank)

	server.Heist = NewHeist(server, player)
	server.Heist.Interaction = i
	server.Heist.Planned = true

	err := heistMessage(s, i, "plan")
	if err != nil {
		log.Error("Unable to create the `Plan Heist` message, error:", err)
	}

	server.Heist.Timer = newWaitTimer(s, i, time.Until(server.Heist.StartTime), startHeist)

	store.Store.Save(HEIST, server.ID, server)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> joinHeist")
	defer log.Debug("<-- joinHeist")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	if contains(server.Heist.Crew, player.ID) {
		discmsg.SendEphemeralResponse(s, i, "You are already a member of the "+theme.Heist+".")
		return
	}
	if msg, ok := heistChecks(server, i, player, server.Targets); !ok {
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}
	if server.Heist.Started {
		discmsg.SendEphemeralResponse(s, i, "The heist has already been started")
		return
	}

	server.Heist.Crew = append(server.Heist.Crew, player.ID)
	err := heistMessage(s, server.Heist.Interaction, "join")
	if err != nil {
		log.Error("Unable to update the heist message, error:", err)
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	economy.WithdrawCredits(bank, account, int(server.Config.HeistCost))
	economy.SaveBank(bank)

	msg := p.Sprintf("You have joined the %s at a cost of %d credits.", theme.Heist, server.Config.HeistCost)
	discmsg.SendEphemeralResponse(s, i, msg)

	store.Store.Save(HEIST, server.ID, server)
}

// leaveHeist attempts to leave a heist previously joined
func leaveHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> leaveHeist")
	defer log.Debug("<-- leaveHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		log.Error("There should be a heist, server:", server.ID, ", heist:", server.Heist)
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if server.Heist.Planner == player.ID {
		discmsg.SendEphemeralResponse(s, i, "You can't leave the "+theme.Heist+", as you are the planner.")
		return
	}
	if !contains(server.Heist.Crew, player.ID) {
		discmsg.SendEphemeralResponse(s, i, "You aren't a member of the "+theme.Heist+".")
		return
	}

	discmsg.SendEphemeralResponse(s, i, "You have left the "+theme.Heist+".")
	server.Heist.Crew = remove(server.Heist.Crew, player.ID)

	err := heistMessage(s, server.Heist.Interaction, "leave")

	if err != nil {
		log.Error("Unable to update the heist message, error:", err)
	}

	store.Store.Save(HEIST, server.ID, server)
}

// cancelHeist cancels a heist that is being planned but has not yet started
func cancelHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> cancelHeist")
	defer log.Debug("<-- cancelHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	if i.Member.User.ID != server.Heist.Planner {
		discmsg.SendEphemeralResponse(s, i, "You cannot cancel the "+theme.Heist+" as you are not the planner.")
		return
	}
	if server.Heist.Started {
		discmsg.SendEphemeralResponse(s, i, "The "+theme.Heist+" has already started and can't be cancelled.")
		return
	}

	err := heistMessage(s, server.Heist.Interaction, "cancel")
	if err != nil {
		log.Error("Unable to mark the heist message as cancelled, error:", err)
	}
	server.Heist.Timer.cancel()
	server.Heist = nil

	discmsg.SendEphemeralResponse(s, i, "The "+theme.Heist+" has been cancelled.")

	store.Store.Save(HEIST, server.ID, server)
}

// startHeist is called once the wait time for planning the heist completes
func startHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> startHeist")
	defer log.Debug("<-- startHeist")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	bank := economy.GetBank(server.ID)
	if server.Heist == nil {
		s.ChannelMessageSend(i.ChannelID, "Error: no heist found.")
		heistMessage(s, i, "cancel")
		return
	}
	if len(server.Targets) == 1 {
		discmsg.SendEphemeralResponse(s, i, "There are no heist targets. Add one using the `/target add` command.")
		server.Heist = nil
		return
	}

	channel := newChannelMute(s, i)
	channel.muteChannel()
	defer channel.unmuteChannel()

	server.Heist.Started = true
	server.Heist.Planned = false

	err := heistMessage(s, i, "start")
	if err != nil {
		log.Error("Unable to mark the heist message as started, error:", err)
	}
	if len(server.Heist.Crew) <= 1 {
		heistMessage(s, i, "ended")
		msg := p.Sprintf("You tried to rally a %s, but no one wanted to follow you. The %s has been cancelled.", theme.Crew, theme.Heist)
		s.ChannelMessageSend(i.ChannelID, msg)
		server.Heist = nil
		return
	}
	msg := p.Sprintf("Get ready! The %s is starting.", theme.Heist)
	s.ChannelMessageSend(i.ChannelID, msg)
	time.Sleep(3 * time.Second)
	heistMessage(s, i, "start")
	target := getTarget(server.Heist, server.Targets)
	results := getHeistResults(server, target)
	msg = p.Sprintf("The %s has decided to hit **%s**.", theme.Crew, target.ID)
	s.ChannelMessageSend(i.ChannelID, msg)
	time.Sleep(3 * time.Second)

	// Process the results
	for _, result := range results.memberResults {
		msg = p.Sprintf(result.message+"\n", result.player.Name)
		s.ChannelMessageSend(i.ChannelID, msg)
		time.Sleep(3 * time.Second)
	}

	if len(results.survivingCrew) == 0 {
		msg = "No one made it out safe."
		s.ChannelMessageSend(i.ChannelID, msg)
	} else {
		// Render the results into a table and returnt he results.
		var tableBuffer strings.Builder
		table := tablewriter.NewWriter(&tableBuffer)
		table.SetHeader([]string{"Player", "Credits"})
		for _, result := range results.survivingCrew {
			data := []string{result.player.Name, p.Sprintf("%d", result.stolenCredits+result.bonusCredits)}
			table.Append(data)
		}
		table.Render()
		s.ChannelMessageSend(i.ChannelID, "```\n"+tableBuffer.String()+"```")
	}

	// Update the status for each player and then save the information
	for _, result := range results.memberResults {
		player := result.player
		if result.status == APPREHENDED || result.status == DEAD {
			handleHeistFailure(server, player, result)
		} else {
			player.Spree++
		}
		if result.stolenCredits != 0 {
			account := bank.GetAccount(player.ID, player.Name)
			economy.DepositCredits(bank, account, result.stolenCredits+result.bonusCredits)
		}
	}
	economy.SaveBank(bank)

	heistMessage(s, i, "ended")

	// Update the heist status information
	server.Config.AlertTime = time.Now().Add(server.Config.PoliceAlert)
	server.Heist = nil
	store.Store.Save(HEIST, server.ID, server)
}

// playerStats shows a player's heist stats
func playerStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> playerStats")
	defer log.Debug("<-- playerStats")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))

	p := getPrinter(i)

	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)

	var sentence string
	if player.Status == APPREHENDED {
		if player.JailTimer.Before(time.Now()) {
			sentence = "Served"
		} else {
			timeRemaining := time.Until(player.JailTimer)
			sentence = format.Duration(timeRemaining)
		}
	} else {
		sentence = "None"
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       player.Name,
			Description: player.CriminalLevel.String(),
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  player.Status,
					Inline: true,
				},
				{
					Name:   "Spree",
					Value:  p.Sprintf("%d", player.Spree),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Bail),
					Value:  p.Sprintf("%d", player.BailCost),
					Inline: true,
				},
				{
					Name:   caser.String(theme.OOB),
					Value:  strconv.FormatBool(player.OOB),
					Inline: true,
				},
				{
					Name:   caser.String(theme.Sentence),
					Value:  sentence,
					Inline: true,
				},
				{
					Name:   APPREHENDED,
					Value:  p.Sprintf("%d", player.JailCounter),
					Inline: true,
				},
				{
					Name:   "Total Deaths",
					Value:  p.Sprintf("%d", player.Deaths),
					Inline: true,
				},
				{
					Name:   "Lifetime Apprehensions",
					Value:  p.Sprintf("%d", player.TotalJail),
					Inline: true,
				},
				{
					Name:   "Credits",
					Value:  p.Sprintf("%d", account.Balance),
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

	p := getPrinter(i)

	var playerID string
	options := i.ApplicationCommandData().Options[0].Options
	for _, option := range options {
		if option.Name == "id" {
			playerID = strings.TrimSpace(option.StringValue())
		}
	}

	server := GetServer(servers, i.GuildID)
	initiatingPlayer := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(initiatingPlayer.ID, initiatingPlayer.Name)

	var player *Player
	if playerID != "" {
		var ok bool
		player, ok = server.Players[playerID]
		if !ok {
			discmsg.SendEphemeralResponse(s, i, "Player "+playerID+" does not exist.")
			return
		}
	} else {
		player = initiatingPlayer
	}

	if player.Status != APPREHENDED || player.OOB {
		var msg string
		if player.ID == i.Member.User.ID {
			msg = "You are not in jail"
		} else {
			msg = p.Sprintf("%s is not in jail", player.Name)
		}
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}
	if player.Status == APPREHENDED && player.JailTimer.Before(time.Now()) {
		discmsg.SendEphemeralResponse(s, i, "You have already served your sentence. Use `/heist release` to be released from jail.")
		return
	}
	if account.Balance < int(player.BailCost) {
		msg := p.Sprintf("You do not have enough credits to play the bail of %d", player.BailCost)
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	economy.WithdrawCredits(bank, account, int(player.BailCost))
	economy.SaveBank(bank)
	player.OOB = true
	store.Store.Save(HEIST, server.ID, server)

	var msg string
	if player.ID == initiatingPlayer.ID {
		msg = "Congratulations, you are now free! Enjoy your freedom while it lasts."
		discmsg.SendEphemeralResponse(s, i, msg)
	} else {
		msg = p.Sprintf("Congratulations, %s, %s bailed you out and now you are free!. Enjoy your freedom while it lasts.", player.Name, initiatingPlayer.Name)
		discmsg.SendResponse(s, i, msg)
	}
}

// releasePlayer releases a player from jail if their sentence has been served.
func releasePlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> releasePlayer")
	defer log.Debug("<-- releasePlayer")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	theme := themes[server.Config.Theme]

	if player.Status != APPREHENDED || player.OOB {
		if player.OOB && player.JailTimer.Before(time.Now()) {
			player.ClearJailAndDeathStatus()
			store.Store.Save(HEIST, server.ID, server)
			discmsg.SendEphemeralResponse(s, i, "You are no longer on probation! 3x penalty removed.")
			return
		}
		discmsg.SendEphemeralResponse(s, i, "I can't remove you from jail if you're not *in* jail")
		return
	}
	if player.JailTimer.After(time.Now()) {
		remainingTime := time.Until(player.JailTimer)
		msg := p.Sprintf("You still have time on your %s, you still need to wait %s.", theme.Sentence, format.Duration(remainingTime))
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	msg := "You served your time. Enjoy the fresh air of freedom while you can."
	if player.OOB {
		msg += "/nYou are no longer on probabtion! 3x penalty removed."
	}

	player.ClearJailAndDeathStatus()
	store.Store.Save(HEIST, server.ID, server)

	discmsg.SendEphemeralResponse(s, i, msg)
}

// revivePlayer raises a player from the dead if their death timer has expired.
func revivePlayer(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> revivePlayer")
	defer log.Debug("<-- revivePlayer")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if player.Status != DEAD {
		discmsg.SendEphemeralResponse(s, i, "You still have a pulse. I can't reive someone who isn't dead.")
		return
	}
	if player.DeathTimer.After(time.Now()) {
		remainingTime := time.Until(player.DeathTimer)
		msg := p.Sprintf("You can't revive yet. You need to wait %s", format.Duration(remainingTime))
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	player.ClearJailAndDeathStatus()
	store.Store.Save(HEIST, server.ID, server)

	discmsg.SendEphemeralResponse(s, i, "You have risen from the dead!")
}

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> resetHeist")
	defer log.Debug("<-- resetHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil || !server.Heist.Planned {
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is being planned.")
		return
	}

	if server.Heist.Timer != nil {
		server.Heist.Timer.cancel()
	}
	heistMessage(s, server.Heist.Interaction, "cancel")
	server.Heist = nil

	if server.Heist == nil || !server.Heist.Planned {
		discmsg.SendResponse(s, i, "The "+theme.Heist+" has been reset.")
	}

	store.Store.Save(HEIST, server.ID, server)
}

// addTarget adds a target for heists
func addTarget(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> addTarget")
	defer log.Debug("<-- addTarget")

	server := GetServer(servers, i.GuildID)

	var id string
	var crewSize, vaultMax, vaultCurrent int64
	var success float64
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "id":
			id = strings.TrimSpace(option.StringValue())
		case "crew":
			crewSize = option.IntValue()
		case "success":
			success = float64(option.IntValue())
		case "vault":
			vaultMax = option.IntValue()
		case "current":
			vaultCurrent = option.IntValue()
		}
	}
	if vaultCurrent == 0 {
		vaultCurrent = vaultMax
	}

	_, ok := server.Targets[id]
	if ok {
		discmsg.SendEphemeralResponse(s, i, "Target \""+id+"\" already exists.")
		return
	}
	for _, target := range server.Targets {
		if target.CrewSize == crewSize {
			discmsg.SendEphemeralResponse(s, i, "Target \""+target.ID+"\" has the same max crew size.")
			return
		}

	}

	target := NewTarget(id, crewSize, success, vaultCurrent, vaultMax)
	server.Targets[target.ID] = target

	discmsg.SendResponse(s, i, "You have added target "+target.ID+" to the new heist.")

	store.Store.Save(HEIST, server.ID, server)
}

// editTarget edits the target information.
func editTarget(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> editTarget")
	defer log.Debug("<-- editTarget")

	var id string
	var crew, vault, current int64
	var success float64
	options := i.ApplicationCommandData().Options[0].Options[0].Options[0].Options
	for _, option := range options {
		switch option.Name {
		case "id":
			id = option.StringValue()
		case "crew":
			crew = option.IntValue()
		case "success":
			success = float64(option.IntValue())
		case "vault":
			vault = option.IntValue()
		case "current":
			current = option.IntValue()
		}
	}

	server := GetServer(servers, i.GuildID)
	target, ok := server.Targets[id]
	if !ok {
		discmsg.SendEphemeralResponse(s, i, "Target \""+id+"\" not found.")
		return
	}
	for _, t := range server.Targets {
		if t.CrewSize == crew && t.ID != target.ID {
			discmsg.SendEphemeralResponse(s, i, "The crew size is not unique; target \""+id+"\" was not updated.")
			return
		}
	}

	if crew != 0 {
		target.CrewSize = crew
	}
	if vault != 0 {
		target.VaultMax = vault
	}
	if current != 0 {
		target.Vault = current
	}
	if success != 0.0 {
		target.Success = success
	}

	discmsg.SendResponse(s, i, "Target \""+id+"\" updated.")

	store.Store.Save(HEIST, server.ID, server)
}

// removeTarget deletes a target.
func removeTarget(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> deleteTarget")
	defer log.Debug("<-- deleteTarget")

	targetID := i.ApplicationCommandData().Options[0].Options[0].Options[0].StringValue()

	server := GetServer(servers, i.GuildID)
	_, ok := server.Targets[targetID]
	if !ok {
		discmsg.SendEphemeralResponse(s, i, "Target \""+targetID+"\" not found.")
		return
	}
	delete(server.Targets, targetID)

	discmsg.SendResponse(s, i, "Target \""+targetID+"\" removed.")

	store.Store.Save(HEIST, server.ID, server)
}

// listTargets displays a list of available heist targets.
func listTargets(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listTargets")
	defer log.Debug("<-- listTargets")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	if len(server.Targets) == 0 {
		msg := "There aren't any targets! To create a target use `/target add`."
		discmsg.SendEphemeralResponse(s, i, msg)
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

		data := []string{target.ID, p.Sprintf("%d", target.CrewSize), p.Sprintf("%d", target.Vault), p.Sprintf("%d", target.VaultMax), p.Sprintf("%.2f", target.Success)}
		table.Append(data)
	}
	table.Render()

	discmsg.SendResponse(s, i, "```\n"+tableBuffer.String()+"\n```")
}

// clearMember clears the criminal state of the player.
func clearMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> clearMember")
	log.Debug("<-- clearMember")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		discmsg.SendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	memberID := i.ApplicationCommandData().Options[0].StringValue()
	server := GetServer(servers, i.GuildID)
	player, ok := server.Players[memberID]
	if !ok {
		discmsg.SendEphemeralResponse(s, i, "Player \""+memberID+"\" not found.")
		return
	}
	player.Reset()
	discmsg.SendResponse(s, i, "Player \""+player.Name+"\"'s settings cleared.")

	store.Store.Save(HEIST, server.ID, server)
}

// listThemes returns the list of available themes that may be used for heists
func listThemes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> listThemes")
	defer log.Debug("<-- listThemes")
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		discmsg.SendEphemeralResponse(s, i, "You are not allowed to use this command.")
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
		discmsg.SendEphemeralResponse(s, i, "You are not allowed to use this command.")
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
		discmsg.SendEphemeralResponse(s, i, "Theme `"+themeName+"` is already being used.")
		return
	}
	theme, err := GetTheme(themeName)
	if err != nil {
		r := []rune(err.Error())
		r[0] = unicode.ToUpper(r[0])
		str := string(r)
		discmsg.SendEphemeralResponse(s, i, str)
		return
	}
	server.Config.Theme = theme.ID
	log.Debug("Now using theme", server.Config.Theme)

	discmsg.SendResponse(s, i, "Theme "+themeName+" is now being used.")

	store.Store.Save(HEIST, server.ID, server)
}

// configCost sets the cost to plan or join a heist
func configCost(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configCost")
	defer log.Debug("<-- configCost")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	cost := options[0].IntValue()
	server.Config.HeistCost = cost

	discmsg.SendResponse(s, i, p.Sprintf("Cost set to %d", cost))

	store.Store.Save(HEIST, server.ID, server)
}

// configSentence sets the base aprehension time when a player is apprehended.
func configSentence(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configSentence")
	defer log.Debug("<-- configSentence")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	sentence := i.ApplicationCommandData().Options[0].Options[0].IntValue()
	server.Config.SentenceBase = time.Duration(sentence * int64(time.Second))

	discmsg.SendResponse(s, i, p.Sprintf("Sentence set to %d", sentence))

	store.Store.Save(HEIST, server.ID, server)
}

// configPatrol sets the time authorities will prevent a new heist following one being completed.
func configPatrol(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configPatrol")
	defer log.Debug("<-- configPatrol")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	patrol := options[0].IntValue()
	server.Config.PoliceAlert = time.Duration(patrol * int64(time.Second))

	discmsg.SendResponse(s, i, p.Sprintf("Patrol set to %d", patrol))

	store.Store.Save(HEIST, server.ID, server)
}

// configBail sets the base cost of bail.
func configBail(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configBail")
	defer log.Debug("<-- configBail")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	bail := options[0].IntValue()
	server.Config.BailBase = bail

	discmsg.SendResponse(s, i, p.Sprintf("Bail set to %d", bail))

	store.Store.Save(HEIST, server.ID, server)
}

// configDeath sets how long players remain dead.
func configDeath(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configDeath")
	defer log.Debug("<-- configDeath")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	death := options[0].IntValue()
	server.Config.PoliceAlert = time.Duration(death * int64(time.Second))

	discmsg.SendResponse(s, i, p.Sprintf("Death set to %d", death))

	store.Store.Save(HEIST, server.ID, server)
}

// configWait sets how long players wait for others to join the heist.
func configWait(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configWait")
	defer log.Debug("<-- configWait")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	wait := options[0].IntValue()
	server.Config.WaitTime = time.Duration(wait * int64(time.Second))

	discmsg.SendResponse(s, i, p.Sprintf("Wait set to %d", wait))

	store.Store.Save(HEIST, server.ID, server)
}

// configPayday sets how many credits a player gets for a playday. This is kinda a hack as
// the configuration is in heist and not in payday, which should one day be fixed.
func configPayday(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configPayday")
	defer log.Debug("<-- configPayday")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	options := i.ApplicationCommandData().Options[0].Options[0].Options
	amount := options[0].IntValue()
	payday.SetPaydayAmount(server.ID, amount)

	discmsg.SendResponse(s, i, p.Sprintf("Payday is set to %d", amount))

	store.Store.Save(HEIST, server.ID, server)
}

// configInfo returns the configuration for the Heist bot on this server.
func configInfo(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Debug("--> configInfo")
	defer log.Debug("<-- configInfo")

	p := getPrinter(i)

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		discmsg.SendEphemeralResponse(s, i, "You are not allowed to use this command.")
		return
	}

	server := GetServer(servers, i.GuildID)

	embed := &discordgo.MessageEmbed{
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "bail",
				Value:  p.Sprintf("%d", server.Config.BailBase),
				Inline: true,
			},
			{
				Name:   "cost",
				Value:  p.Sprintf("%d", server.Config.HeistCost),
				Inline: true,
			},
			{
				Name:   "death",
				Value:  p.Sprintf("%.f", server.Config.DeathTimer.Seconds()),
				Inline: true,
			},
			{
				Name:   "patrol",
				Value:  p.Sprintf("%.f", server.Config.PoliceAlert.Seconds()),
				Inline: true,
			},
			{
				Name:   "payday",
				Value:  p.Sprintf("%d", payday.GetPaydayAmount(server.ID)),
				Inline: true,
			},
			{
				Name:   "sentence",
				Value:  p.Sprintf("%.f", server.Config.SentenceBase.Seconds()),
				Inline: true,
			},
			{
				Name:   "wait",
				Value:  p.Sprintf("%.f", server.Config.WaitTime.Seconds()),
				Inline: true,
			},
		},
	}

	embeds := []*discordgo.MessageEmbed{
		embed,
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Heist Configuration",
			Embeds:  embeds,
		},
	})
	if err != nil {
		log.Error("Unable to send a response, error:", err)
	}
}

// Start initializes anything needed by the heist bot.
func Start(s *discordgo.Session) {
	servers = LoadServers()
	themes = LoadThemes()

	go vaultUpdater()
}

// GetCommands ret urns the component handlers, command handlers, and commands for the Heist bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	commands := make([]*discordgo.ApplicationCommand, 0, len(adminCommands)+len(playerCommands))
	commands = append(commands, adminCommands...)
	commands = append(commands, playerCommands...)
	return componentHandlers, commandHandlers, commands
}
