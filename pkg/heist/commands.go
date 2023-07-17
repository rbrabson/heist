/*
commands contains the list of commands and messages sent to Discord, or commands processed when received from Discord.
*/
package heist

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/joho/godotenv"
	"github.com/rbrabson/heist/pkg/checks"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

// TODO: ensure heist commands are only run in the #heist channel
// TODO: check to see if Heist has been paused (it should be in the state)

var (
	bot   *Bot
	appID string
)

func init() {
	godotenv.Load()
	appID = os.Getenv("APP_ID")
}

// componentHandlers are the buttons that appear on messages sent by this bot.
var (
	componentsHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"join_heist":   joinHeist,
		"leave_heist":  leaveHeist,
		"cancel_heist": cancelHeist,
	}
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"plan":    planHeist,
		"reset":   resetHeist,
		"themes":  listThemes,
		"theme":   setTheme,
		"version": version,
		"clear":   clearMember,
		"targets": listTargets,
	}

	commands = []*discordgo.ApplicationCommand{
		{
			Name:        "plan",
			Description: "Plans a heist",
		},
		{
			Name:        "reset",
			Description: "Resets a heist",
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
		},
		{
			Name:        "targets",
			Description: "Gets the list of available heist targets",
		},
		{
			Name:        "themes",
			Description: "Gets the list of available heist themes",
		},
		{
			Name:        "theme",
			Description: "Sets the current heist theme",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "name",
					Description: "Name of the theme to set",
					Required:    true,
				},
			},
		},
		{
			Name:        "version",
			Description: "Returns the version of heist running on the server",
		},
	}
)

/******** UTILITY FUNCTIONS ********/

// getAssignedRoles returns a list of discord roles assigned to the user
func getAssignedRoles(s *discordgo.Session, i *discordgo.InteractionCreate) discordgo.Roles {
	guild, err := s.Guild(i.GuildID)
	if err != nil {
		log.Error("Error:", err)
		return nil
	}

	member, err := s.GuildMember(i.GuildID, i.Member.User.ID)
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
	log.Info("--> commandFailure")
	defer log.Info("<-- commandFailure")

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
func heistMessage(s *discordgo.Session, i *discordgo.InteractionCreate, action string) error {
	log.Info("--> heistMessage")
	defer log.Info("<-- heistMessage")

	server := bot.servers.Servers[i.GuildID]
	player := server.Players[i.Member.User.ID]
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
	log.Info("--> planHeist")
	defer log.Info("<-- planHeist")

	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	if server.Heist != nil {
		bot.Session.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "A " + server.Theme.Heist + " is already being planned.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		return
	}

	player := getPlayer(server, i.Member.User.ID, i.Member.User.Username)

	server.Heist = NewHeist(player)
	server.Heist.Interaction = i
	server.Heist.Planned = true

	err := heistMessage(s, i, "plan")
	if err != nil {
		log.Fatal(err)
	}

	server.Heist.Timer = newWaitTimer(s, i, server.Config.WaitTime, startHeist)

	file, err := json.MarshalIndent(bot.servers, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	store := store.NewStore()
	store.SaveHeistState(file)
}

// joinHeist attempts to join a heist that is being planned
func joinHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> joinHeist")
	defer log.Info("<-- joinHeist")

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
	err = heistMessage(s, server.Heist.Interaction, "join")
	if err != nil {
		log.Fatal(err)
	}
}

// leaveHeist attempts to leave a heist previously joined
func leaveHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> leaveHeist")
	defer log.Info("<-- leaveHeist")

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

	err = heistMessage(s, server.Heist.Interaction, "leave")
	if err != nil {
		log.Error(err)
	}
}

// cancelHeist cancels a heist that is being planned but has not yet started
func cancelHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> cancelHeist")
	defer log.Info("<-- cancelHeist")

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
	if i.Member.User.ID != server.Heist.Planner {
		commandFailure(s, i, "You cannot cancel the "+server.Theme.Heist+" as you are not the planner.")
		return
	}

	heistMessage(s, server.Heist.Interaction, "cancel")
	server.Heist.Timer.cancel()
	server.Heist = nil

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "The " + server.Theme.Heist + " has been cancelled.",
		},
	})
	if err != nil {
		log.Error(err)
	}
}

// startHeist is called once the wait time for planning the heist completes
func startHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> startHeist")
	defer log.Info("<-- startHeist")

	server := bot.servers.Servers[s.State.Application.GuildID]
	err := heistMessage(s, i, "start")
	if err != nil {
		log.Fatal(err)
	}

	// TODO: start the game.

	// For now, just clear out the heist so we can continue....
	time.Sleep(5 * time.Second)
	err = s.ChannelMessageDelete(i.ChannelID, i.Message.ID)
	if err != nil {
		log.Fatal(err)
	}
	server.Heist = nil
}

/******** ADMIN COMMANDS ********/

// Reset resets the heist in case it hangs
func resetHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> resetHeist")
	defer log.Info("<-- resetHeist")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		return
	}
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	if server.Heist == nil || !server.Heist.Planned {
		err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "No " + server.Theme.Heist + " is being planned.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Error(err)
		}
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
				Content: "The " + server.Theme.Heist + " has been reset.",
				Flags:   discordgo.MessageFlagsEphemeral,
			},
		})
		if err != nil {
			log.Error(err)
		}
	}
}

// listTargets displays a list of available heist targets.
func listTargets(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> listTargets")
	defer log.Info("<-- listTargets")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		return
	}

	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	if len(server.Targets) == 0 {
		msg := "There aren't any targets! To create a target use `/createtarget`."
		commandFailure(s, i, msg)
		return
	}

	var targets, crews, vaults strings.Builder
	for _, target := range server.Targets {
		targets.WriteString(target.ID + "\n")
		crews.WriteString(strconv.Itoa(target.CrewSize) + "\n")
		vaults.WriteString(strconv.Itoa(target.Vault) + "\n")
	}

	caser := cases.Caser(cases.Title(language.Und, cases.NoLower))
	embeds := []*discordgo.MessageEmbed{
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
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// clearMember clears the criminal state of the player.
func clearMember(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> clearMember")
	log.Info("<-- clearMember")

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
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	player, ok := server.Players[memberID]
	if !ok {
		commandFailure(s, i, "Player not found.")
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
		log.Fatal(err)
	}
}

// listThemes returns the list of available themes that may be used for heists
func listThemes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		log.Info("User is not an administrator")
		return
	}

	themes, err := GetThemes()
	if err != nil {
		return
	}

	embeds := []*discordgo.MessageEmbed{
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
	}

	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Embeds: embeds,
			Flags:  discordgo.MessageFlagsEphemeral,
		},
	})
}

// setTheme sets the heist theme to the one specified in the command
func setTheme(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> setTheme")
	defer log.Info("<-- setTheme")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		return
	}

	server := bot.servers.Servers[i.GuildID]
	if server == nil {
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}
	var themeName string
	options := i.ApplicationCommandData().Options
	for _, option := range options {
		if option.Name == "name" {
			themeName = strings.TrimSpace(option.StringValue())
		}
	}

	if themeName == server.Config.Theme {
		commandFailure(s, i, "Theme "+themeName+" is already being used.")
		return
	}
	theme, err := LoadTheme(themeName)
	if err != nil {
		commandFailure(s, i, "Theme "+themeName+" does not exist.")
		return
	}
	server.Config.Theme = themeName
	server.Theme = *theme

	err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "Theme " + themeName + " is now being used.",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// version shows the version of heist you are running.
func version(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Info("--> version")
	defer log.Info("<-- version")

	if !checks.IsAdminOrServerManager(getAssignedRoles(s, i)) {
		return
	}
	server, ok := bot.servers.Servers[i.GuildID]
	if !ok {
		log.Info("Getting new server")
		server = NewServer(i.GuildID)
		bot.servers.Servers[server.ID] = server
	}

	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "You are running Heist version " + server.Config.Version + ".",
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		log.Fatal(err)
	}
}

// addBotCommands adds all commands that may be issued from a given server.
func addBotCommands(b *Bot) {
	log.Debug("adding bot commands")
	bot = b

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

	bot.Session.ApplicationCommandBulkOverwrite(appID, "", commands)
}
