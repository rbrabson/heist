/*
commands contains the list of commands and messages sent to Discord, or commands processed when received from Discord.
*/
package heist

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/rbrabson/heist/pkg/checks"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

// TODO: ensure heist commands are only run in the #heist channel
// TODO: check to see if Heist has been paused (it should be in the state)

var (
	bot *Bot
)

// componentHandlers are the buttons that appear on messages sent by this bot.
var (
	componentsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"join_heist":   joinHeist,
		"leave_heist":  leaveHeist,
		"cancel_heist": cancelHeist,
	}
)

/******** UTILITY FUNCTIONS ********/

// getAssignedRoles returns a list of discord roles assigned to the user
func getAssignedRoles(s *discordgo.Session, m *discordgo.MessageCreate) discordgo.Roles {
	guild, err := s.Guild(m.GuildID)
	if err != nil {
		log.Error("Error:", err)
		return nil
	}

	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		log.Error(err)
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

// getPlayer returns the player with the given ID on the server. If the player doesn't
// exist, a new player with the ID and name provided is created and added to the server.
func getPlayer(server *Server, playerID string, playerName string) *Player {
	player, ok := server.Players[playerID]
	if !ok {
		player = NewPlayer(playerID, playerName)
		server.Players[player.ID] = player
	}
	player.Name = playerName
	return player
}

/******** MESSAGE UTILITIES ********/

// commandFailure is a utility routine used to send an error response to a user's reaction to a bot's message.
func commandFailure(s *discordgo.Session, i *discordgo.InteractionCreate, msg string) {
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: msg,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// heistMessage sends the main command used to plan, join and leave a heist. It also handles the case where
// the heist starts, disabling the buttons to join/leave/cancel the heist.
func heistMessage(s *discordgo.Session, server *Server, player *Player, channelID string, messageID string, planning bool) (*discordgo.Message, error) {
	var status string
	var buttonDisabled bool
	if planning {
		timestamp := fmt.Sprintf("<t:%v:R> ", time.Now().Add(server.Config.WaitTime).Unix())
		status = "Starts " + timestamp
		buttonDisabled = false
	} else {
		status = "Started"
		buttonDisabled = true
	}

	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	embeds := []*discordgo.MessageEmbed{
		{
			Type:        discordgo.EmbedTypeRich,
			Title:       "Heist",
			Description: "A new " + server.Theme.Heist + " is being planned by " + player.Name + ". You can join the " + server.Theme.Heist + " at any time prior to the " + server.Theme.Heist + " starting.",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Status",
					Value:  status,
					Inline: true,
				},
				{
					Name:   "Number of " + caser.String(server.Theme.Crew) + "  Members",
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

	var message *discordgo.Message
	var err error
	if messageID == "" {
		msg := &discordgo.MessageSend{
			Embeds:     embeds,
			Components: components,
		}

		message, err = s.ChannelMessageSendComplex(channelID, msg)
	} else {
		msg := &discordgo.MessageEdit{
			Embeds:     embeds,
			Components: components,
			Channel:    channelID,
			ID:         messageID,
		}
		message, err = s.ChannelMessageEditComplex(msg)
	}
	if err != nil {
		return nil, err
	}

	return message, nil
}

/******** PLAYER COMMANDS ********/

// planHeist plans a new heist
func planHeist(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != bot.prefix+" plan" {
		return
	}

	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server = NewServer(m.GuildID)
		bot.servers.Servers[m.GuildID] = server
		log.Info(m.GuildID)
	}
	if server.Heist != nil {
		reply, _ := bot.Session.ChannelMessageSendReply(m.ChannelID, "A "+server.Theme.Heist+" is already being planned.", m.Reference())
		time.Sleep(3 * time.Second)
		bot.Session.ChannelMessageDelete(m.ChannelID, reply.ID)
		bot.Session.ChannelMessageDelete(m.ChannelID, m.ID)
		return
	}

	player := getPlayer(server, m.Author.ID, m.Author.Username)

	server.Heist = NewHeist(player)
	server.Heist.Planned = true

	planner := server.Players[server.Heist.Planner]
	message, err := heistMessage(s, server, planner, m.ChannelID, "", true)
	if err != nil {
		log.Fatal(err)
	}

	server.Heist.MessageID = message.ID
	server.Heist.Timer = newWaitTimer(s, server, m.ChannelID, message.ID, server.Config.WaitTime, startHeist)

	file, err := json.MarshalIndent(bot.servers, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	store := store.NewStore()
	store.SaveHeistState(file)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
		commandFailure(s, i, "No "+server.Theme.Heist+" could be found.")
	}
	if server.Heist == nil {
		commandFailure(s, i, "No "+server.Theme.Heist+" is planned.")
		return
	}
	player := getPlayer(server, i.Member.User.ID, i.Member.User.Username)
	if contains(server.Heist.Crew, player.ID) {
		commandFailure(s, i, "You are already a member of the "+server.Theme.Heist+".")
		return
	}
	var err error

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have joined the " + server.Theme.Heist + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	server.Heist.Crew = append(server.Heist.Crew, player.ID)
	planner := server.Players[server.Heist.Planner]
	_, err = heistMessage(s, server, planner, i.ChannelID, i.Message.ID, true)
	if err != nil {
		log.Fatal(err)
	}
}

// leaveHeist attempts to leave a heist previously joined
func leaveHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
		commandFailure(s, i, "No "+server.Theme.Heist+" could be found.")
	}
	if server.Heist == nil {
		commandFailure(s, i, "No "+server.Theme.Heist+" is planned.")
		return
	}
	player := getPlayer(server, i.Member.User.ID, i.Member.User.Username)

	if server.Heist.Planner == player.ID {
		commandFailure(s, i, "You can't leave the "+server.Theme.Heist+", as you are the planner.")
		return
	}

	if !contains(server.Heist.Crew, player.ID) {
		commandFailure(s, i, "You aren't a member of the "+server.Theme.Heist+".")
		return
	}
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You have left the " + server.Theme.Heist + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Error(err)
	}
	server.Heist.Crew = remove(server.Heist.Crew, player.ID)

	planner := server.Players[server.Heist.Planner]
	_, err = heistMessage(s, server, planner, i.ChannelID, i.Message.ID, true)
	if err != nil {
		log.Error(err)
	}
}

// cancelHeist cancels a heist that is being planned but has not yet started
func cancelHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
		commandFailure(s, i, "No "+server.Theme.Heist+" could be found.")
		return
	}
	if server.Heist == nil {
		commandFailure(s, i, "No "+server.Theme.Heist+" is planned.")
		return
	}

	// Need to save the author of the person who is planning the heist
	if i.Member.User.ID != server.Heist.Planner {
		commandFailure(s, i, "You cannot cancel the "+server.Theme.Heist+" as you are not the planner.")
		return
	}
	err := s.ChannelMessageDelete(i.Message.ChannelID, i.Message.ID)
	if err != nil {
		log.Error(err)
	}
	server.Heist.Timer.cancel()
	server.Heist = nil

	s.ChannelMessageSend(i.ChannelID, "The "+server.Theme.Heist+" has been cancelled.")
}

// startHeist is called once the wait time for planning the heist completes
func startHeist(s *discordgo.Session, server *Server, channelID string, messageID string) {
	planner := server.Players[server.Heist.Planner]
	_, err := heistMessage(s, server, planner, channelID, messageID, false)
	if err != nil {
		log.Fatal(err)
	}

	// TODO: start the game.

	// For now, just clear out the heist so we can continue....
	time.Sleep(5 * time.Second)
	err = s.ChannelMessageDelete(channelID, messageID)
	if err != nil {
		log.Fatal(err)
	}
	server.Heist = nil
}

func heistHelp(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != bot.prefix+" help" && m.Content != bot.prefix {
		return
	}

	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server = NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
	}

	var playerCommands strings.Builder
	playerCommands.WriteString("help: a summary of available commands\n")
	playerCommands.WriteString("plan: plan a new " + server.Theme.Heist + "\n")
	embedFields := []*discordgo.MessageEmbedField{
		{
			Name:   "Commands",
			Value:  playerCommands.String(),
			Inline: true,
		},
	}
	if checks.IsAdmin(getAssignedRoles(s, m)) {
		var adminCommands strings.Builder
		adminCommands.WriteString("clear: clears jail and death statuses\n")
		adminCommands.WriteString("reset: resets a hung " + server.Theme.Heist + "\n")
		adminCommands.WriteString("targets: lists available targets\n")
		adminCommands.WriteString("theme: sets the theme\n")
		adminCommands.WriteString("themes: lists available themes\n")
		adminCommands.WriteString("version: shows the version of heist\n")
		embedFields = append(embedFields, &discordgo.MessageEmbedField{
			Name:   "Admin Commands",
			Value:  adminCommands.String(),
			Inline: false,
		})
	}

	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeRich,
				Title:       "Available Commands",
				Description: "Available Commands for the Heist bot",
				Fields:      embedFields,
			},
		},
	}

	s.ChannelMessageSendComplex(m.ChannelID, msg)
}

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != bot.prefix+" reset" {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}
	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server := NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
		bot.Session.ChannelMessageSendReply(m.ChannelID, "No "+server.Theme.Heist+" is being planned.", m.Reference())
	}
	if server.Heist == nil || !server.Heist.Planned {
		bot.Session.ChannelMessageSendReply(m.ChannelID, "No "+server.Theme.Heist+" is being planned.", m.Reference())
		return
	}

	if server.Heist.Timer != nil {
		server.Heist.Timer.cancel()
	}
	s.ChannelMessageDelete(m.ChannelID, server.Heist.MessageID)
	server.Heist = nil

	s.ChannelMessageSendReply(m.ChannelID, "The "+server.Theme.Heist+" has been reset.", m.Reference())
}

// clearMember clears the criminal state of the player.
func listTargets(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandPrefix := bot.prefix + " targets"
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != commandPrefix {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}

	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server = NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	if len(server.Targets) == 0 {
		msg := "There aren't any targets! To create a target use `" +
			bot.prefix + " createtarget`."
		s.ChannelMessageSendReply(m.ChannelID, msg, m.Reference())
		return
	}

	var targets, crews, vaults strings.Builder
	for _, target := range server.Targets {
		targets.WriteString(target.ID + "\n")
		crews.WriteString(strconv.Itoa(target.CrewSize) + "\n")
		vaults.WriteString(strconv.Itoa(target.Vault) + "\n")
	}
	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeRich,
				Title:       "Available Targets",
				Description: "Available targets for the Heist bot",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Target",
						Value:  targets.String(),
						Inline: true,
					},
					{
						Name:   "Max Crew",
						Value:  crews.String(),
						Inline: true,
					},
					{
						Name:   caser.String(server.Theme.Vault),
						Value:  vaults.String(),
						Inline: true,
					},
				},
			},
		},
	}
	s.ChannelMessageSendComplex(m.ChannelID, msg)

}

// clearMember clears the criminal state of the player.
func clearMember(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandPrefix := bot.prefix + " clear"
	if m.Author.ID == s.State.User.ID {
		return
	}
	if !strings.HasPrefix(m.Content, commandPrefix) {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}
	memberID := strings.TrimSpace(m.Content[len(commandPrefix):])
	if memberID == "" {
		s.ChannelMessageSendReply(m.ChannelID, "Usage: "+bot.prefix+" clear <userID>.", m.Reference())
		return
	}
	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server := NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	player, ok := server.Players[memberID]
	if !ok {
		s.ChannelMessageSendReply(m.ChannelID, "Player not found.", m.Reference())
	}
	player.ClearSettings()
	s.ChannelMessageSendReply(m.ChannelID, "Player settings cleared.", m.Reference())
}

// listThemes returns the list of available themes that may be used for heists
func listThemes(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != bot.prefix+" themes" {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}
	themes, err := GetThemes()
	if err != nil {
		return
	}
	msg := &discordgo.MessageSend{
		Embeds: []*discordgo.MessageEmbed{
			{
				Type:        discordgo.EmbedTypeRich,
				Title:       "Available Themes",
				Description: "Available Themes for the Heist bot",
				Fields: []*discordgo.MessageEmbedField{
					{
						Name:   "Themes",
						Value:  strings.Join(themes[:], ","),
						Inline: true,
					},
				},
			},
		},
	}

	s.ChannelMessageSendComplex(m.ChannelID, msg)
}

// setTheme sets the heist theme to the one specified in the command
func setTheme(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandPrefix := bot.prefix + " theme"
	if m.Author.ID == s.State.User.ID {
		return
	}
	if !strings.HasPrefix(m.Content, commandPrefix) {
		return
	}
	if m.Content == bot.prefix+" themes" {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}
	server := bot.servers.Servers[m.GuildID]
	if server == nil {
		server = NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	themeName := strings.TrimSpace(m.Content[len(commandPrefix):])
	if themeName == "" {
		s.ChannelMessageSend(m.ChannelID, "Usage: "+bot.prefix+" theme <theme_name>.")
		return
	}
	if themeName == server.Config.Theme {
		s.ChannelMessageSend(m.ChannelID, "Theme "+themeName+" is already being used.")
		return
	}
	theme, err := LoadTheme(themeName)
	if err != nil {
		s.ChannelMessageSend(m.ChannelID, "Theme "+themeName+" does not exist.")
		return
	}
	server.Config.Theme = themeName
	server.Theme = *theme
	s.ChannelMessageSend(m.ChannelID, "Theme "+themeName+" is now being used.")
}

// version shows the version of heist you are running.
func version(s *discordgo.Session, m *discordgo.MessageCreate) {
	commandPrefix := bot.prefix + " version"
	if m.Author.ID == s.State.User.ID {
		return
	}
	if m.Content != commandPrefix {
		return
	}
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, m)) {
		return
	}
	server, ok := bot.servers.Servers[m.GuildID]
	if !ok {
		server := NewServer(m.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	bot.Session.ChannelMessageSendReply(m.ChannelID, "You are running heist version "+server.Config.Version+".", m.Reference())
}

// addBotCommands adds all commands that may be issued from a given server.
func addBotCommands(b *Bot) {
	log.Debug("adding bot commands")
	var err error
	bot = b

	bot.Session.AddHandler(func(s *discordgo.Session, r *discordgo.Ready) {
		log.Info("Heist bot is up!")
	})
	// Components are part of interactions, so we register InteractionCreate handler
	bot.Session.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionMessageComponent:
			if h, ok := componentsHandlers[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}
		}
	})
	if err != nil {
		log.Fatalf("Cannot add the component handlers: %v", err)
	}

	bot.Session.AddHandler(listThemes)
	bot.Session.AddHandler(planHeist)
	bot.Session.AddHandler(resetHeist)
	bot.Session.AddHandler(heistHelp)
	bot.Session.AddHandler(setTheme)
	bot.Session.AddHandler(clearMember)
	bot.Session.AddHandler(listTargets)
	bot.Session.AddHandler(version)
}
