package race

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/format"
	"github.com/rbrabson/heist/pkg/math"
	"github.com/rbrabson/heist/pkg/msg"
	log "github.com/sirupsen/logrus"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

var (
	session *discordgo.Session
)

var (
	racers = []string{
		"race_bet_one",
		"race_bet_two",
		"race_bet_three",
		"race_bet_four",
		"race_bet_five",
		"race_bet_six",
		"race_bet_seven",
		"race_bet_eight",
		"race_bet_nine",
		"race_bet_ten",
		"race_bet_eleven",
	}
)

var (
	componentHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"join_race":       joinRace,
		"race_bet_one":    betOnRace,
		"race_bet_two":    betOnRace,
		"race_bet_three":  betOnRace,
		"race_bet_four":   betOnRace,
		"race_bet_five":   betOnRace,
		"race_bet_six":    betOnRace,
		"race_bet_seven":  betOnRace,
		"race_bet_eight":  betOnRace,
		"race_bet_nine":   betOnRace,
		"race_bet_ten":    betOnRace,
		"race_bet_eleven": betOnRace,
	}

	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"race":       race,
		"race-admin": admin,
	}

	playerCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "race",
			Description: "Race game commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "start",
					Description: "Starts a new race.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
				{
					Name:        "stats",
					Description: "Returns the race stats for the player.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	adminCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "race-admin",
			Description: "Race game admin commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "reset",
					Description: "Resets a hung race.",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}
)

/******** MESSAGE UTILITIES ********/

// getPrinter returns a printer for the given locale of the user initiating the message.
func getPrinter(i *discordgo.InteractionCreate) *message.Printer {
	log.Trace("--> getPrinter")
	defer log.Trace("<-- getPrinter")

	tag, err := language.Parse(string(i.Locale))
	if err != nil {
		log.Error("Unable to parse locale, error:", err)
		tag = language.English
	}
	return message.NewPrinter(tag)
}

// getRacerButtons returns action rows for the buttons used to vote on the racers.
func getRacerButtons(race *Race) []discordgo.ActionsRow {
	log.Trace("--> getRacerButtons")
	defer log.Trace("<-- getRacerButtons")

	buttonsPerRow := 5
	rows := make([]discordgo.ActionsRow, 0, len(race.Racers)/buttonsPerRow)

	racersIncludedInButtons := 0
	for len(race.Racers) > racersIncludedInButtons {
		racersNotInButtons := len(race.Racers) - racersIncludedInButtons
		buttonsForNextRow := math.Min(buttonsPerRow, racersNotInButtons)
		buttons := make([]discordgo.MessageComponent, 0, buttonsForNextRow)
		for j := 0; j < buttonsForNextRow; j++ {
			index := j + racersIncludedInButtons
			button := discordgo.Button{
				Label:    race.Racers[index].Player.Name,
				Style:    discordgo.PrimaryButton,
				CustomID: racers[index],
			}
			buttons = append(buttons, button)
		}
		racersIncludedInButtons += buttonsForNextRow

		row := discordgo.ActionsRow{Components: buttons}
		rows = append(rows, row)
		log.WithFields(log.Fields{
			"numRacers": len(race.Racers),
			"buttons":   len(buttons),
			"row":       len(rows),
		}).Trace("Race Buttons")
	}

	return rows
}

// raceMessage sends the main command used to start and join the race. It also handles the case where
// the race begins, disabling the buttons to join the race.
func raceMessage(s *discordgo.Session, i *discordgo.InteractionCreate, action string) error {
	log.Trace("--> raceMessage")
	defer log.Trace("<-- raceMessage")

	p := getPrinter(i)

	server := GetServer(i.GuildID)
	race := server.Race
	racerNames := make([]string, 0, len(race.Racers))
	for _, racer := range race.Racers {
		racerNames = append(racerNames, racer.Player.Name)
	}

	var msg string
	if action == "start" || action == "join" || action == "update" {
		until := time.Until(race.StartTime)
		msg = p.Sprintf(":triangular_flag_on_post: A race is starting! Click the button to join the race! :triangular_flag_on_post:\n\t\t\t\t\tThe race will begin in %s!", format.Duration(until))
	} else if action == "betting" {
		until := time.Until(race.BetEndTime)
		msg = p.Sprintf(":triangular_flag_on_post: The racers have been set - betting is now open! :triangular_flag_on_post:\n\t\tYou have %s to place a %d credit bet!", format.Duration(until), server.Config.BetAmount)
	} else if action == "started" {
		msg = ":checkered_flag: The race is now in progress! :checkered_flag:"
	} else if action == "ended" {
		msg = ":checkered_flag: The race has ended - lets find out the results. :checkered_flag:"
	} else if action == "cancelled" {
		msg = "Not enough players entered the race, so it was cancelled."
	} else {
		errMsg := fmt.Sprintf("Unrecognized action: %s", action)
		log.Error(errMsg)
		return errors.New(errMsg)
	}

	embeds := []*discordgo.MessageEmbed{
		{
			Type:  discordgo.EmbedTypeRich,
			Title: "Race",
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   msg,
					Inline: false,
				},
				{
					Name:   p.Sprintf("Racers (%d)", len(race.Racers)),
					Value:  strings.Join(racerNames, ", "),
					Inline: false,
				},
			},
		},
	}

	var err error
	if action == "start" {
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join",
					Style:    discordgo.SuccessButton,
					CustomID: "join_race",
				},
			}},
		}
		err = s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Embeds:     embeds,
				Components: components,
			},
		})
	} else if action == "join" {
		components := []discordgo.MessageComponent{
			discordgo.ActionsRow{Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Join",
					Style:    discordgo.SuccessButton,
					CustomID: "join_race",
				},
			}},
		}
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds,
			Components: &components,
		})
	} else if action == "update" {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds: &embeds,
		})
	} else if action == "betting" {
		components := []discordgo.MessageComponent{}
		rows := getRacerButtons(race)
		for _, row := range rows {
			components = append(components, row)
		}
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds,
			Components: &components,
		})
	} else {
		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Embeds:     &embeds,
			Components: &[]discordgo.MessageComponent{},
		})
	}

	return err
}

// sendRaceResults sends the results of a race to the Discord server
func sendRaceResults(s *discordgo.Session, channelID string, server *Server) {
	log.Trace("--> sendRaceResults")
	defer log.Trace("<-- sendRaceResults")

	p := message.NewPrinter(language.English)
	racers := server.Race.Racers
	raceResults := make([]*discordgo.MessageEmbedField, 0, 4)
	raceResults = append(raceResults, &discordgo.MessageEmbedField{
		Name:   p.Sprintf(":first_place: %s", racers[0].Player.Name),
		Value:  p.Sprintf("%s\n%.2fs\nPrize: %d", racers[0].Character.Emoji, racers[0].Speed, racers[0].Prize),
		Inline: true,
	})
	raceResults = append(raceResults, &discordgo.MessageEmbedField{
		Name:   p.Sprintf(":second_place: %s", racers[1].Player.Name),
		Value:  p.Sprintf("%s\n%.2fs\nPrize: %d", racers[1].Character.Emoji, racers[1].Speed, racers[1].Prize),
		Inline: true,
	})
	if len(racers) >= 3 {
		raceResults = append(raceResults, &discordgo.MessageEmbedField{
			Name:   p.Sprintf(":third_place: %s", racers[2].Player.Name),
			Value:  p.Sprintf("%s\n%.2fs\nPrize: %d", racers[2].Character.Emoji, racers[2].Speed, racers[2].Prize),
			Inline: true,
		})
	}

	betWinners := make([]string, 0, 1)
	for _, bet := range server.Race.Bets {
		if bet.Racer == racers[0] {
			betWinners = append(betWinners, bet.Name)
		}
	}
	var winners string
	if len(betWinners) > 0 {
		winners = strings.Join(betWinners, "\n")
	} else {
		winners = "No one guessed the winner."
	}
	betEarnings := server.Config.BetAmount * len(server.Race.Racers)
	betResults := &discordgo.MessageEmbedField{
		Name:   p.Sprintf("Bet earnings of %d", betEarnings),
		Value:  winners,
		Inline: false,
	}
	raceResults = append(raceResults, betResults)
	embeds := []*discordgo.MessageEmbed{
		{
			Title:  "Race Results",
			Fields: raceResults,
		},
	}
	s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
		Embeds: embeds,
	})
}

/******** COMMAND ROUTERS ********/

// race routes the various `race` subcommands to the appropriate handlers.
func race(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> race")
	defer log.Trace("<-- race")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "start":
		prepareRace(s, i)
	case "stats":
		raceStats(s, i)
	}
}

// admin routes various `race-admin` subcommands to the appropriate handlers.
func admin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> admin")
	defer log.Trace("<-- admin")

	options := i.ApplicationCommandData().Options
	switch options[0].Name {
	case "reset":
		resetRace(s, i)
	}
}

/******** PLAYER COMMANDS ********/

// prepareRace starts a race that other members may join.
func prepareRace(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> prepareRace")
	defer log.Trace("<-- prepareRace")

	p := message.NewPrinter(language.English)
	server := GetServer(i.GuildID)

	server.mutex.Lock()
	if server.Race != nil {
		msg.SendEphemeralResponse(s, i, "A race is already starting. You can join that race instead.")
		server.mutex.Unlock()
		return
	}
	timeSinceLastRace := time.Since(server.LastRaceEnded)
	if timeSinceLastRace < server.Config.WaitBetweenRaces {
		timeUntilRaceCanStart := server.Config.WaitBetweenRaces - timeSinceLastRace
		msg.SendEphemeralResponse(s, i, p.Sprintf("The racers are resting. Try again in %s!", format.Duration(timeUntilRaceCanStart)))
		server.mutex.Unlock()
		return
	}

	server.Race = NewRace(server)
	server.Race.Planned = true
	server.Race.Interaction = i

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	mode := Modes[server.Config.Mode]
	racer := NewRacer(player, mode)
	server.Race.Racers = append(server.Race.Racers, racer)

	err := raceMessage(s, i, "start")
	if err != nil {
		log.Error("Unable to update the race message, error:", err)
	}
	log.WithFields(log.Fields{
		"Name":      player.Name,
		"ID":        player.ID,
		"Character": racer.Character.Emoji,
	}).Debug("Start Race")
	server.mutex.Unlock()

	// Update the message every five seconds with the new expiration time until the
	// time has expired.
	for !time.Now().After(server.Race.StartTime) {
		maximumWait := time.Until(server.Race.StartTime)
		timeToWait := math.Min(maximumWait, 5*time.Second)
		if timeToWait < 0 {
			break
		}
		time.Sleep(timeToWait)
		err = raceMessage(s, i, "update")
		if err != nil {
			log.Error("Unable to update the time for the race message, error:", err)
		}
	}

	startRace(s, i)
}

// startRace is called once the timer waiting for players to join the race or place
// bets expires.
func startRace(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> startRace")
	defer log.Trace("<-- startRace")

	server := GetServer(i.GuildID)

	server.mutex.Lock()
	if len(server.Race.Racers) < server.Config.MinRacers {
		log.WithFields(log.Fields{"Number": len(server.Race.Racers)}).Info("Race cancelled due to lack of racers.")
		raceMessage(s, i, "cancelled")
		server.Race = nil
		server.mutex.Unlock()
		return
	}

	err := raceMessage(s, i, "betting")
	if err != nil {
		log.Error("Unable to update the race message for betting, error:", err)
	}
	server.mutex.Unlock()

	// Update the message every five seconds with the new expiration time until the
	// time has expired.
	//
	// FOR NOW: try to leave the channels un-muted and see how it goes. Uncomment the
	//          following three lines if it gets unruly.
	// mute := channel.NewChannelMute(s, i)
	// mute.MuteChannel()
	// defer mute.UnmuteChannel()
	for !time.Now().After(server.Race.BetEndTime) {
		maximumWait := time.Until(server.Race.BetEndTime)
		timeToWait := math.Min(maximumWait, 5*time.Second)
		if timeToWait < 0 {
			break
		}
		time.Sleep(timeToWait)
		err = raceMessage(s, i, "betting")
		if err != nil {
			log.Error("Unable to update the time for the race message, error:", err)
		}
	}

	server.mutex.Lock()
	err = raceMessage(s, i, "started")
	if err != nil {
		log.Error("Unable to update the race message for the race starting, error:", err)
	}
	server.Race.Started = true
	server.Race.Planned = false
	server.mutex.Unlock()

	server.RunRace(i.ChannelID)

	err = raceMessage(s, i, "ended")
	if err != nil {
		log.Error("Unable to update the race message, error:", err)
	}

	raceMessage(s, i, "ended")

	calculateRacerWinnings(server)
	calcualteBetWinnings(server)

	bank := economy.GetBank(i.GuildID)

	for index, racer := range server.Race.Racers {
		racer.Player.NumRaces++
		switch index {
		case 0:
			racer.Player.Results.Win++
		case 1:
			racer.Player.Results.Place++
		case 2:
			racer.Player.Results.Show++
		default:
			racer.Player.Results.Losses++
		}
		if racer.Prize != 0 && racer.Player.ID != "" {
			account := bank.GetAccount(racer.Player.ID, racer.Player.Name)
			account.DepositCredits(racer.Prize)
			racer.Player.Results.Earnings += racer.Prize
		}
	}
	for _, bet := range server.Race.Bets {
		player := server.Players[bet.ID]
		player.Results.BetsPlaced++
		if bet.Winnings != 0 {
			player.Results.BetsWon++
			player.Results.BetEarnings += bet.Winnings
			account := bank.GetAccount(bet.ID, bet.Name)
			account.DepositCredits(bet.Winnings)
		}
	}
	economy.SaveBank(bank)

	sendRaceResults(s, i.ChannelID, server)
	server.GamesPlayed++
	server.LastRaceEnded = time.Now()
	server.Race = nil
	SaveServer(server)
}

// joinRace attempts to join a race that is getting ready to start.
func joinRace(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> joinRace")
	defer log.Trace("<-- joinRace")

	p := message.NewPrinter(language.English)

	server := GetServer(i.GuildID)
	mode := Modes[server.Config.Mode]

	server.mutex.Lock()
	defer server.mutex.Unlock()

	if server.Race == nil {
		msg.SendEphemeralResponse(s, i, "No race is planned.")
		return
	}
	if !server.Race.Planned {
		msg.SendEphemeralResponse(s, i, "The race has already started, so you can't join.")
		return
	}
	for _, racer := range server.Race.Racers {
		if i.Member.User.ID == racer.Player.ID {
			msg.SendEphemeralResponse(s, i, "You are already a member of the race.")
			return
		}
	}
	if server.Config.MaxRacers == len(server.Race.Racers) {
		resp := p.Sprintf("You can't join the race, as there are already %d entered into the race.", server.Config.MaxRacers)
		msg.SendEphemeralResponse(s, i, resp)
	}

	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)
	racer := NewRacer(player, mode)
	server.Race.Racers = append(server.Race.Racers, racer)
	err := raceMessage(s, server.Race.Interaction, "join")
	if err != nil {
		log.Error("Unable to update the race message, error:", err)
	}
	log.WithFields(log.Fields{
		"Name":      player.Name,
		"ID":        player.ID,
		"Character": racer.Character.Emoji,
	}).Debug("Join Race")
	msg.SendEphemeralResponse(s, i, "You have joined the race.")
}

// raceStats returns a players race stats.
func raceStats(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> joinRace")
	defer log.Trace("<-- joinRace")

	p := getPrinter(i)
	server := GetServer(i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	var betPercentage float64
	if player.Results.BetsPlaced > 0 {
		betPercentage = 100 * float64(player.Results.BetsWon) / float64(player.Results.BetsPlaced)
	} else {
		betPercentage = 0.0
	}
	embeds := []*discordgo.MessageEmbed{
		{
			Type:  discordgo.EmbedTypeRich,
			Title: player.Name,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "First",
					Value:  p.Sprintf("%d (%.0f%%)", player.Results.Win, 100*float64(player.Results.Win)/float64(player.NumRaces)),
					Inline: true,
				},
				{
					Name:   "Second",
					Value:  p.Sprintf("%d (%.0f%%)", player.Results.Place, 100*float64(player.Results.Place)/float64(player.NumRaces)),
					Inline: true,
				},
				{
					Name:   "Third",
					Value:  p.Sprintf("%d (%.0f%%)", player.Results.Show, 100*float64(player.Results.Show)/float64(player.NumRaces)),
					Inline: true,
				},
				{
					Name:   "Losses",
					Value:  p.Sprintf("%d (%.0f%%)", player.Results.Losses, 100*float64(player.Results.Losses)/float64(player.NumRaces)),
					Inline: true,
				},
				{
					Name:   "Races",
					Value:  p.Sprintf("%d (%.0f%%)", player.NumRaces, 100*float64(player.NumRaces)/float64(server.GamesPlayed)),
					Inline: true,
				},
				{
					Name:   "Earnings",
					Value:  p.Sprintf("%d", player.Results.Earnings),
					Inline: true,
				},
				{
					Name:   "Bets Won",
					Value:  p.Sprintf("%d (%.0f%%)", player.Results.BetsWon, betPercentage),
					Inline: true,
				},
				{
					Name:   "Bet Earnings",
					Value:  p.Sprintf("%d", player.Results.BetEarnings),
					Inline: true,
				},
				{
					Name:   "Net Bet Earnings",
					Value:  p.Sprintf("%d", player.Results.BetEarnings-player.Results.BetsPlaced*server.Config.BetAmount),
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

// betOnRace processes a bet placed by a member on the race.
func betOnRace(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> betOnRace")
	defer log.Trace("<-- betOnRace")

	server := GetServer(i.GuildID)
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username, i.Member.Nick)

	if server.Race.Started || server.Race.Ended {
		msg.SendEphemeralResponse(s, i, "You can't place a bet after the race has started.")
		return
	}
	for _, bettor := range server.Race.Bets {
		if bettor.ID == i.Member.User.ID {
			msg.SendEphemeralResponse(s, i, "You have already bet on the race.")
			return
		}
	}
	bank := economy.GetBank(server.ID)
	account := bank.GetAccount(player.ID, player.Name)
	if account.CurrentBalance < int(server.Config.BetAmount) {
		msg.SendEphemeralResponse(s, i, "You don't have enough money to cover the bet.")
		return
	}
	racer := server.Race.getRacer(i.Interaction.MessageComponentData().CustomID)
	if racer == nil {
		msg.SendEphemeralResponse(s, i, "Racer could not be found.")
		return
	}
	bettor := &Bettor{
		ID:    player.ID,
		Name:  player.Name,
		Racer: racer,
		Bet:   server.Config.BetAmount,
	}
	server.Race.Bets = append(server.Race.Bets, bettor)
	account.WithdrawCredits(bettor.Bet)
	economy.SaveBank(bank)
	log.WithFields(log.Fields{
		"Name":  player.Name,
		"ID":    player.ID,
		"Racer": racer.Player.Name,
	}).Debug("Placed Bet")

	resp := fmt.Sprintf("You placed a %d %s bet on %s", server.Config.BetAmount, server.Config.Currency, racer.Player.Name)
	msg.SendEphemeralResponse(s, i, resp)
}

/******** ADMIN COMMANDS ********/

// resetRace resets a hung race.
func resetRace(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> resetRace")
	defer log.Trace("<-- resetRace")

	server := GetServer(i.GuildID)
	server.Race = nil
	// Uncomment this out if we change to mute the channel again
	// mute := channel.NewChannelMute(s, i)
	// mute.UnmuteChannel()
	msg.SendResponse(s, i, "The race has been reset.")
}

// GetCommands ret urns the component handlers, command handlers, and commands for the Race game.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	commands := make([]*discordgo.ApplicationCommand, 0, len(adminCommands)+len(playerCommands))
	commands = append(commands, adminCommands...)
	commands = append(commands, playerCommands...)
	return componentHandlers, commandHandlers, commands
}
