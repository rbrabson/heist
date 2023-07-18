package heist

import (
	"os"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

type Bot struct {
	Session *discordgo.Session
	timer   chan int
}

func NewBot() *Bot {
	token := os.Getenv("BOT_TOKEN")
	s, _ := discordgo.New("Bot " + token)

	// TODO: load the system state

	bot := &Bot{
		Session: s,
		timer:   make(chan int),
	}
	bot.Session.Identify.Intents = discordgo.IntentsAllWithoutPrivileged
	addBotCommands(bot)

	log.Debug(servers)

	go bot.vaultUpdater()

	return bot
}

type number interface {
	int | int32 | int64 | float32 | float64
}

func min[N number](v1 N, v2 N) N {
	if v1 < v2 {
		return v1
	}
	return v2
}

func (b *Bot) vaultUpdater() {
	const timer = time.Duration(120 * time.Second)
	time.Sleep(20 * time.Second)
	for {
		for _, server := range servers.Servers {
			for _, target := range server.Targets {
				vault := min(target.Vault+(target.VaultMax*4/100), target.VaultMax)
				target.Vault = vault
			}
		}
		time.Sleep(timer)
	}
}

func (b *Bot) subtractCosts(player *Player, cost uint) {
	// get the bank
	// bank.WithdrawCredits(player, cost)
}

func (b *Bot) accountCheck(author *Player) {
	// if author is not in the list of players
	//   create a new player and add them
}

func (b *Bot) requirementCheck(author *Player) {
	// Verify there is at least one target
	// Verify there is no heist underway
	// Verify the player isn't already in the crew
	// Verify player.Status == "Apprehended"
	//   if remainingTime == "No Cooldown"
	//		tell user to use !heist release to get released from jail
	//   else
	//      tell user they are serving, how long they have to wait, and cost to get out
	//   return failure w/ the message
	// Verify playerStatus == "Dead"
	//	  If "No Cooldown"
	//        tell user they can use !heist revive to revive themself
	//    else
	//         tell user they are still dead and can use !heist revive when timer expires
	//    return failure w/ the message
	// Verify the user does not have enough credits
	//    tell user they don't have enough creditw to cover the cost of entry, and how much they need
	//    return failure w/ the message
	// Verify the alert level == "Hot"
	//     tell the user the police are on high alert, and to try again when the patrol time exxpires (get time remaining)
	// return success
}

func (b *Bot) scheduleHeist(s *discordgo.Session, i *discordgo.InteractionCreate) {
	server, ok := servers.Servers[i.GuildID]
	if !ok {
		server = NewServer(i.GuildID)
		servers.Servers[i.GuildID] = server
	}
	player := server.GetPlayer(i.Member.User.ID, i.Member.User.Username)
	server.Heist.Planned = true
	server.Heist.Planner = player.ID
	server.Heist.Crew = append(server.Heist.Crew, server.Heist.Planner)
	b.timer = make(chan int)

	t := time.NewTimer(server.Config.WaitTime)
	select {
	case <-t.C:
		b.startHeist()
	case <-b.timer:
		b.cancelHeist(i.GuildID)
		if !t.Stop() {
			<-t.C
		}
	}

	// If, after wait time, crew <= 1 then cancel the heist with error message
	// Else, start the heist
}

func (b *Bot) cancelHeist(guildID string) {
	log.Info("cancelling heist")

	server, ok := servers.Servers[guildID]
	if !ok {
		server = NewServer(guildID)
	}

	server.Heist = nil
}

func (b *Bot) startHeist() {
	log.Info("starting heist")
	// Lock the channel
	// Config.HeistStart = true
	// get the game outcome
	// Delete the heist planning message
	// Message that indicates the heist is starting
	// await self.bot.say("Get ready! The {} is starting with {}\nThe {} has decided to "
	//                    "hit **{}**.".format(t_heist, start_output, t_crew, target))
	// sleep for 3 seconds
	// Unlock the channel
	// display the results. If anyone escaped, then:
	//  Players    Credits Obtained   Bonuses    Total
	// else
	//  "No one made it out safe."
	// resetHeist
}

/*

// Clear resets a member's jail or death status
func (bot *Bot) Clear(m *discordgo.MessageCreate) error {
	roles := bot.getAssignedRoles(m)
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	log.Printf("%s administratevely cleared %s\n", "name of person clearing", "name of person cleared")

	return nil
}

// Version returns the version of heist that is running
func (bot *Bot) Version(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	log.Printf("You are running Heist version %s.", "version")

	return nil
}

// Targets shows a list of targets
func (bot *Bot) Targets(roles []*discordgo.Role) error {
	// Get the list of targets from the state
	// If the length of the targets == 0, then no targets.
	//    tell the user to create a target using the !heist command
	// Else
	//   get the target name
	//   get the crew
	//   get the success
	//   get the valut
	//   sort the list based on some value (name? not sure)
	//   format as a table & return it

	return nil
}

// Bailout allows you to pay for the release of someone. The default is yourself.
func (bot *Bot) Bailout(roles []*discordgo.Role) error {
	// Get the name of "Bail" from the current theme
	// Get the name of the "Sentence" from the current theme
	// If user is not provided
	//    player == author
	// Else
	//    player is the specified user
	//
	// If user's status is not "Apprehended"
	//    log.Printf("%s is not in jail.\n", player.name)
	//    return
	// Get the bail cost. If < amount in the player's bank
	//    log.Printf("You do not have enough to afford the %v amount.\n", bailAmount)
	// If bailing yourself out
	//     Prompt to see if user wants to try to make some money
	// Else
	//     Prompt if the user wants to pay the bail amount
	// Wait for 15 seconds for the response. If no response
	//     log.Info("You took too long. canceling transaction.")
	//      return
	// If response is "Yes"
	//     log.Prinf(""Congratulations {}, you are free! Enjoy your freedom while it lasts...".format(player.display_name))")
	//     reduce the amount from the player's bank
	//     save the system state
	// Else if "No"
	//     log.Info("Canceling transaction.")
	// Else
	//     log.Info("Incorrect response, canceling transaction")
	return nil
}

// CreateTarget adds a target to heist
func (bot *Bot) CreateTarget(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	// Lots here. Read through the requirements....

	return nil
}

// EditTarget edits a heist target
func (bot *Bot) EditTarget(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	// Lots here. Read through the requirements

	return nil
}

// RemoveTarget removes a target from the heist list
func (bot *Bot) RemoveTarget(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	// Lots here. Read through the requirements

	return nil
}

// Info shows the Heist settings for this server
func (bot *Bot) Info(roles []*discordgo.Role) error {
	return nil
}

// Release removes you from jain or clears bail status if the sentence is served.
func (bot *Bot) Release(roles []*discordgo.Role) error {
	return nil
}

// Revive revives you from the dead
func (bot *Bot) Revive(roles []*discordgo.Role) error {
	return nil
}

// Stats shows your Heist stats
func (bot *Bot) Stats(roles []*discordgo.Role) error {
	return nil
}

// Play begins a Heist.
func (bot *Bot) Play(roles []*discordgo.Role) error {
	return nil
}

// GrandHeist begins a Grand Heist
func (bot *Bot) GrandHeist(roles []*discordgo.Role) error {
	if !checks.IsAdmin(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Pause pauses the heist play
func (bot *Bot) Pause(roles []*discordgo.Role) error {
	if !checks.IsAllowed(roles, botCommander) {
		return ErrNotAllowed
	}

	return nil
}

// Mention mentions the @Heist role
func (bot *Bot) Mention(roles []*discordgo.Role) error {
	if !checks.IsAllowed(roles, botCommander) {
		return ErrNotAllowed
	}

	return nil
}

// SetHeist sets different options in the Heist config.
func (bot *Bot) SetHeist(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// SetTheme sets the theme for the heist
func (bot *Bot) SetTheme(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Output changes how detailed the starting output is.
// - None: Displays just the number of crew members.
// - Short: Displays five participants and truncates the rest
// - Long: Shows the entire crew list. WARNING: not suitable for really big crews.
func (bot *Bot) Output(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Sent3ence sets the base apprehension time when caught.
func (bot *Bot) Sentence(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// cost sets the cost to play Heist.
func (bot *Bot) Cost(roles []*discordgo.Role) error {
	if !checks.IsAdminOrServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Authorities sets the time authorities will prevent heists.
func (bot *Bot) Authorities(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Bail sets the base cost of bail.
func (bot *Bot) Bail(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Death sets how long players are dead.
func (bot *Bot) Death(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Hardcore sets the game to hardcore mode. Death will wipe credits and chips.
func (bot *Bot) Hardcore(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// Wait sets how long a player can gatheer other players for a heist.
func (bot *Bot) Wait(roles []*discordgo.Role) error {
	if !checks.IsServerManager(roles) {
		return ErrNotAllowed
	}

	return nil
}

// ShowResults shows the results of the heist.
func (bot *Bot) ShowResults(roles []*discordgo.Role) error {

	return nil
}
*/
