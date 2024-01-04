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

	"github.com/rbrabson/heist/pkg/channel"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/cogs/payday"
	"github.com/rbrabson/heist/pkg/format"
	hmath "github.com/rbrabson/heist/pkg/math"
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
	servers   map[string]*Server
	themes    map[string]*Theme
	targetSet map[string]*Targets
)

// componentHandlers are the buttons that appear on messages sent by this bot.
var (
	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"join_heist": joinHeist,
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
					Name:        "stats",
					Description: "Shows a user's stats.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "start",
					Description: "Plans a new heist.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "targets",
					Description: "Gets the list of available heist targets.",
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
	log.Trace("--> config")
	defer log.Trace("<-- config")

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

// theme routes the theme commands to the proper handlers.
func theme(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> theme")
	defer log.Trace("<-- theme")

	options := i.ApplicationCommandData().Options[0].Options
	switch options[0].Name {
	case "list":
		listThemes(s, i)
	case "set":
		setTheme(s, i)
	}
}

/******** UTILITY FUNCTIONS ********/

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
	log.Trace("--> heistMessage")
	defer log.Trace("<-- heistMessage")

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

	server.Heist.Mutex.Lock()
	crew := make([]string, 0, len(server.Heist.Crew))
	for _, id := range server.Heist.Crew {
		crew = append(crew, server.Players[id].Name)
	}
	server.Heist.Mutex.Unlock()

	theme := themes[server.Config.Theme]
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	msg := p.Sprintf("A new %s is being planned by %s. You can join the %s for a cost of %d credits at any time prior to the %s starting.", theme.Heist, player.Name, theme.Heist, server.Config.HeistCost, theme.Heist)
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
					Name:   p.Sprintf("%s (%d members)", caser.String(theme.Crew), len(crew)),
					Value:  strings.Join(crew, ", "),
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
	log.Trace("--> admin")
	defer log.Trace("<-- admin")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "clear":
		clearMember(s, i)
	case "config":
		config(s, i)
	case "reset":
		resetHeist(s, i)
	case "theme":
		theme(s, i)
	}
}

// heist routes the commands to the subcommand and subcommandgroup handlers
func heist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> heist")
	defer log.Trace("<-- heist")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "bail":
		bailoutPlayer(s, i)
	case "start":
		planHeist(s, i)
	case "stats":
		playerStats(s, i)
	case "targets":
		listTargets(s, i)
	}
}

/******** PLAYER COMMANDS ********/

// planHeist plans a new heist“
func planHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> planHeist")
	defer log.Trace("<-- planHeist")

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	server.Mutex.Lock()
	// Heist is already in progress
	if server.Heist != nil {
		discmsg.SendEphemeralResponse(s, i, "A "+theme.Heist+" is already being planned.")
		server.Mutex.Unlock()
		return
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	// Basic error checks for the heist
	msg, ok := heistChecks(server, i, player, server.Targets)
	if !ok {
		discmsg.SendEphemeralResponse(s, i, msg)
		server.Mutex.Unlock()
		return
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	account.WithdrawCredits(int(server.Config.HeistCost))
	economy.SaveBank(bank)

	server.Heist = NewHeist(server, player)
	server.Heist.Interaction = i
	server.Heist.Planned = true

	err := heistMessage(s, i, "plan")
	if err != nil {
		log.Error("Unable to create the `Plan Heist` message, error:", err)
	}
	server.Mutex.Unlock()

	/*
		if msg != "" {
			discmsg.SendEphemeralResponse(s, i, msg)
		}
	*/

	for !time.Now().After(server.Heist.StartTime) {
		maximumWait := time.Until(server.Heist.StartTime)
		timeToWait := hmath.Min(maximumWait, 5*time.Second)
		if timeToWait < 0 {
			break
		}
		time.Sleep(timeToWait)
		err := heistMessage(s, i, "update")
		if err != nil {
			log.Error("Unable to update the time for the heist message, error:", err)
		}
	}

	startHeist(s, i)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> joinHeist")
	defer log.Trace("<-- joinHeist")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is planned.")
		return
	}
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	server.Heist.Mutex.Lock()
	isMember := contains(server.Heist.Crew, player.ID)
	server.Heist.Mutex.Unlock()
	if isMember {
		discmsg.SendEphemeralResponse(s, i, "You are already a member of the "+theme.Heist+".")
		return
	}
	msg, ok := heistChecks(server, i, player, server.Targets)
	if !ok {
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}
	if server.Heist.Started {
		discmsg.SendEphemeralResponse(s, i, "The heist has already been started")
		return
	}

	server.Heist.Mutex.Lock()
	server.Heist.Crew = append(server.Heist.Crew, player.ID)
	server.Heist.Mutex.Unlock()
	err := heistMessage(s, server.Heist.Interaction, "join")
	if err != nil {
		log.Error("Unable to update the heist message, error:", err)
	}

	// Withdraw the cost of the heist from the player's account. We know the player already
	// as the required number of credits as this is verified in `heistChecks`.
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	account.WithdrawCredits(int(server.Config.HeistCost))
	economy.SaveBank(bank)

	if msg != "" {
		msg := p.Sprintf("%s You have joined the %s at a cost of %d credits.", msg, theme.Heist, server.Config.HeistCost)
		discmsg.SendEphemeralResponse(s, i, msg)
	} else {
		msg := p.Sprintf("You have joined the %s at a cost of %d credits.", theme.Heist, server.Config.HeistCost)
		discmsg.SendEphemeralResponse(s, i, msg)
	}

	store.Store.Save(HEIST, server.ID, server)
}

// startHeist is called once the wait time for planning the heist completes
func startHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> startHeist")
	defer log.Trace("<-- startHeist")

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
		discmsg.SendEphemeralResponse(s, i, "There are no heist targets.")
		server.Heist = nil
		return
	}

	mute := channel.NewChannelMute(s, i)
	mute.MuteChannel()
	defer mute.UnmuteChannel()

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
	log.Debug("Heist is starting")
	msg := p.Sprintf("Get ready! The %s is starting with %d members.", theme.Heist, len(server.Heist.Crew))
	s.ChannelMessageSend(i.ChannelID, msg)
	time.Sleep(3 * time.Second)
	heistMessage(s, i, "start")
	target := getTarget(server.Heist, server.Targets)
	results := getHeistResults(server, target)
	log.Debug("Hitting " + target.ID)
	msg = p.Sprintf("The %s has decided to hit **%s**.", theme.Crew, target.ID)
	s.ChannelMessageSend(i.ChannelID, msg)
	time.Sleep(3 * time.Second)

	// Process the results
	for _, result := range results.memberResults {
		msg = p.Sprintf(result.message+"\n", "**"+result.player.Name+"**")
		if result.status == APPREHENDED {
			msg += p.Sprintf("`%s dropped out of the game.`", result.player.Name)
		}
		s.ChannelMessageSend(i.ChannelID, msg)
		time.Sleep(3 * time.Second)
	}

	if results.escaped == 0 {
		msg = "\nNo one made it out safe."
		s.ChannelMessageSend(i.ChannelID, msg)
	} else {
		msg = "\nThe raid is now over. Distributing player spoils."
		s.ChannelMessageSend(i.ChannelID, msg)
		// Render the results into a table and returnt he results.
		var tableBuffer strings.Builder
		table := tablewriter.NewWriter(&tableBuffer)
		table.SetBorder(false)
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
		table.SetHeader([]string{"Player", "Loot", "Bonus", "Total"})
		for _, result := range results.survivingCrew {
			data := []string{result.player.Name, p.Sprintf("%d", result.stolenCredits), p.Sprintf("%d", result.bonusCredits), p.Sprintf("%d", result.stolenCredits+result.bonusCredits)}
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
		if results.escaped > 0 && result.stolenCredits != 0 {
			account := bank.GetAccount(player.ID, player.Name)
			account.DepositCredits(result.stolenCredits + result.bonusCredits)
			target.Vault -= int64(result.stolenCredits)
			log.WithFields(log.Fields{"Member": account.Name, "Stolen": result.stolenCredits, "Bonus": result.bonusCredits}).Debug("Heist Loot")
		}
	}
	target.Vault = hmath.Max(target.Vault, target.VaultMax*4/100)

	economy.SaveBank(bank)

	heistMessage(s, i, "ended")

	// Update the heist status information
	server.Config.AlertTime = time.Now().Add(server.Config.PoliceAlert)
	server.Heist = nil
	store.Store.Save(HEIST, server.ID, server)
}

// playerStats shows a player's heist stats
func playerStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> playerStats")
	defer log.Trace("<-- playerStats")

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
					Value:  p.Sprintf("%d", account.CurrentBalance),
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
	log.Trace("--> bailoutPlayer")
	log.Trace("<-- bailoutPlayer")

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
		discmsg.SendEphemeralResponse(s, i, "You have already served your sentence.")
		player.Reset()
		return
	}
	if account.CurrentBalance < int(player.BailCost) {
		msg := p.Sprintf("You do not have enough credits to play the bail of %d", player.BailCost)
		discmsg.SendEphemeralResponse(s, i, msg)
		return
	}

	account.WithdrawCredits(int(player.BailCost))
	economy.SaveBank(bank)
	player.OOB = true
	store.Store.Save(HEIST, server.ID, server)

	var msg string
	if player.ID == initiatingPlayer.ID {
		msg = p.Sprintf("Congratulations, you are now free! You spent %d credits on your bail. Enjoy your freedom while it lasts.", player.BailCost)
		discmsg.SendEphemeralResponse(s, i, msg)
	} else {
		msg = p.Sprintf("Congratulations, %s, %s bailed you out by spending %d credits and now you are free!. Enjoy your freedom while it lasts.", player.Name, initiatingPlayer.Name, player.BailCost)
		discmsg.SendResponse(s, i, msg)
	}
}

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> resetHeist")
	defer log.Trace("<-- resetHeist")

	mute := channel.NewChannelMute(s, i)
	defer mute.UnmuteChannel()

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]
	if server.Heist == nil {
		discmsg.SendEphemeralResponse(s, i, "No "+theme.Heist+" is being planned.")
		return
	}

	heistMessage(s, server.Heist.Interaction, "cancel")
	server.Heist = nil
	discmsg.SendResponse(s, i, "The "+theme.Heist+" has been reset.")

	store.Store.Save(HEIST, server.ID, server)
}

// listTargets displays a list of available heist targets.
func listTargets(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> listTargets")
	defer log.Trace("<-- listTargets")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	theme := themes[server.Config.Theme]

	if len(server.Targets) == 0 {
		msg := "There aren't any targets!"
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
	table.SetBorder(false)
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
	table.SetHeader([]string{"ID", "Max Crew", theme.Vault, "Max " + theme.Vault, "Success Rate"})
	for _, target := range targets {
		data := []string{target.ID, p.Sprintf("%d", target.CrewSize), p.Sprintf("%d", target.Vault), p.Sprintf("%d", target.VaultMax), p.Sprintf("%.2f", target.Success)}
		table.Append(data)
	}
	table.Render()

	discmsg.SendEphemeralResponse(s, i, "```\n"+tableBuffer.String()+"\n```")
}

// clearMember clears the criminal state of the player.
func clearMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> clearMember")
	log.Trace("<-- clearMember")

	memberID := i.ApplicationCommandData().Options[0].Options[0].StringValue()
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
	log.Trace("--> listThemes")
	defer log.Trace("<-- listThemes")

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
	log.Trace("--> setTheme")
	defer log.Trace("<-- setTheme")

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
	log.Debug("Now using theme ", server.Config.Theme)

	discmsg.SendResponse(s, i, "Theme "+themeName+" is now being used.")

	store.Store.Save(HEIST, server.ID, server)
}

// configCost sets the cost to plan or join a heist
func configCost(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> configCost")
	defer log.Trace("<-- configCost")

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
	log.Trace("--> configSentence")
	defer log.Trace("<-- configSentence")

	p := getPrinter(i)

	server := GetServer(servers, i.GuildID)
	sentence := i.ApplicationCommandData().Options[0].Options[0].IntValue()
	server.Config.SentenceBase = time.Duration(sentence * int64(time.Second))

	discmsg.SendResponse(s, i, p.Sprintf("Sentence set to %d", sentence))

	store.Store.Save(HEIST, server.ID, server)
}

// configPatrol sets the time authorities will prevent a new heist following one being completed.
func configPatrol(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> configPatrol")
	defer log.Trace("<-- configPatrol")

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
	log.Trace("--> configBail")
	defer log.Trace("<-- configBail")

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
	log.Trace("--> configDeath")
	defer log.Trace("<-- configDeath")

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
	log.Trace("--> configWait")
	defer log.Trace("<-- configWait")

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
	log.Trace("--> configPayday")
	defer log.Trace("<-- configPayday")

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
	log.Trace("--> configInfo")
	defer log.Trace("<-- configInfo")

	p := getPrinter(i)

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
	targetSet = LoadTargets()
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
