package heist

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/format"
	hmath "github.com/rbrabson/heist/pkg/math"
	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
)

const (
	FREE        = "Free"
	DEAD        = "Dead"
	APPREHENDED = "Apprehended"
)

// HeistResult is the result of a heist.
type HeistResult struct {
	escaped       int
	apprehended   int
	dead          int
	memberResults []*HeistMemberResult
	survivingCrew []*HeistMemberResult
	target        *Target
}

// HeistMemberResult is the result for a single player who is a member of the heist crew.
type HeistMemberResult struct {
	player        *Player
	status        string
	message       string
	stolenCredits int
	bonusCredits  int
}

// heistChecks returns an error, with appropriate message, if a heist cannot be started.
func heistChecks(server *Server, i *discordgo.InteractionCreate, player *Player, targets map[string]*Target) (string, bool) {

	p := getPrinter(i)

	theme := themes[server.Config.Theme]
	bank := economy.GetBank(server.ID)

	if len(targets) == 0 {
		msg := "Oh no! There are no targets!"
		return msg, false
	}
	log.Debug("Heist:", server.Heist)
	var isMember bool
	if server.Heist != nil {
		server.Heist.Mutex.Lock()
		isMember = contains(server.Heist.Crew, player.ID)
		server.Heist.Mutex.Unlock()
	}
	if server.Heist != nil && isMember {
		msg := fmt.Sprintf("You are already in the %s.", theme.Crew)
		return msg, false
	}
	account := bank.GetAccount(player.ID, player.Name)
	if account.CurrentBalance < int(server.Config.HeistCost) {
		msg := p.Sprintf("You do not have enough credits to cover the cost of entry. You need %d credits to participate", server.Config.HeistCost)
		return msg, false
	}
	if server.Config.AlertTime.After(time.Now()) {
		remainingTime := time.Until(server.Config.AlertTime)
		msg := p.Sprintf("The %s are on high alert after the last target. We should wait for things to cool off before hitting another target. Time remaining: %s.", theme.Police, format.Duration(remainingTime))
		return msg, false
	}
	if player.Status == APPREHENDED {
		if player.OOB {
			if player.JailTimer.Before(time.Now()) {
				msg := p.Sprintf("Your %s is over, and you are no longer on probation! 3x penalty removed.", theme.Sentence)
				player.ClearJailAndDeathStatus()
				return msg, true
			}
			return "", true
		}
		if player.JailTimer.After(time.Now()) {
			remainingTime := time.Until(player.JailTimer)
			msg := fmt.Sprintf("You are in %s. You are serving a %s of %s.\nYou can wait out your remaining %s of %s, or pay %d credits to be released on %s.",
				theme.Jail, theme.Sentence, format.Duration(player.Sentence), theme.Sentence, format.Duration(remainingTime), player.BailCost, theme.Bail)
			return msg, false
		}
		msg := "You served your time. Enjoy the fresh air of freedom while you can."
		player.ClearJailAndDeathStatus()
		return msg, true
	}
	if player.Status == DEAD {
		if player.DeathTimer.After(time.Now()) {
			remainingTime := time.Until(player.DeathTimer)
			msg := p.Sprintf("You are dead. You will revive in %s", format.Duration(remainingTime))
			return msg, false
		}
		msg := "You have risen from the dead!`."
		player.ClearJailAndDeathStatus()
		return msg, true
	}

	return "", true
}

// calculateCredits determines the number of credits stolen by each surviving crew member.
func calculateCredits(results *HeistResult) {
	log.Trace("--> calculateCredits")
	defer log.Trace("<-- calculateCredits")

	// Take 3/4 of the amount of the vault, and distribute it among those who survived.
	numSurvived := results.escaped + results.apprehended
	stolenPerSurivor := int(math.Round(float64(results.target.Vault) * 0.75 / float64(numSurvived)))
	totalStolen := numSurvived * stolenPerSurivor

	// Get a "base amount" of loot stolen. If you are apprehended, this is what you get. If you escaped you get 2x as much.
	baseStolen := totalStolen / (2*results.escaped + results.apprehended)

	// Caculate a "base amount". Those who escape get 2x those who don't. So Divide the
	log.WithFields(log.Fields{"Target": results.target.ID, "Vault": results.target.Vault, "Survivors": numSurvived, "Base Credits": baseStolen}).Debug("Looted")
	for _, player := range results.survivingCrew {
		if player.status == FREE {
			player.stolenCredits = 2 * baseStolen
		} else {
			player.stolenCredits = baseStolen
		}
	}
}

// calculateBonusRate calculates the bonus amount to add to the success rate
// for a heist. The closer you are to the maximum crew size, the larger
// the bonus amount.
func calculateBonusRate(heist *Heist, target *Target) int {
	log.Trace("--> calculateBonus")
	defer log.Trace("<-- calculateBonus")

	percent := 100 * int64(len(heist.Crew)) / target.CrewSize
	log.WithField("Percent", percent).Debug("Percentage for calculating success bonus")
	if percent <= 20 {
		return 0
	}
	if percent <= 40 {
		return 1
	}
	if percent <= 60 {
		return 3
	}
	if percent <= 80 {
		return 4
	}
	return 5
}

// calculateSuccessRate returns the liklihood of a successful raid for each
// member of the heist crew.
func calculateSuccessRate(heist *Heist, target *Target) int {
	log.Trace("--> calculateSuccessRate")
	defer log.Trace("<-- calculateSuccessRate")

	bonus := calculateBonusRate(heist, target)
	successChance := int(math.Round(target.Success)) + bonus
	log.WithFields(log.Fields{"BonusRate": bonus, "TargetSuccess": math.Round(target.Success), "SuccessChance": successChance}).Debug("Success Rate")
	return successChance
}

// handleHeistFailure updates the status of a player who is apprehended or killed during a heist.
func handleHeistFailure(server *Server, player *Player, result *HeistMemberResult) {
	log.Trace("--> handleHeistFailure")
	defer log.Trace("<-- handleHeistFailure")

	if result.status == APPREHENDED {
		sentence := int64(server.Config.SentenceBase) * (player.JailCounter + 1)
		bail := server.Config.BailBase
		if player.OOB {
			bail *= 3
		}
		player.BailCost = bail
		player.CriminalLevel++
		player.JailCounter++
		player.TotalJail++
		player.OOB = false
		player.Sentence = time.Duration(sentence)
		player.JailTimer = time.Now().Add(player.Sentence)
		player.Spree = 0
		player.Status = APPREHENDED

		log.WithFields(log.Fields{
			"player":        player.Name,
			"bail":          player.BailCost,
			"criminalLevel": player.CriminalLevel,
			"jailCounter":   player.JailCounter,
			"jailTimier":    player.JailTimer,
			"oob":           player.OOB,
			"sentence":      player.Sentence,
			"spree":         player.Spree,
			"status":        player.Status,
			"totalJail":     player.TotalJail,
		}).Debug("Apprehended")

		return
	}

	player.BailCost = 0
	player.CriminalLevel = 0
	player.DeathTimer = time.Now().Add(server.Config.DeathTimer)
	player.JailCounter = 0
	player.JailTimer = time.Time{}
	player.OOB = false
	player.Sentence = 0
	player.Spree = 0
	player.Status = DEAD
	player.Deaths++

	log.WithFields(log.Fields{
		"player":        player.Name,
		"bail":          player.BailCost,
		"criminalLevel": player.CriminalLevel,
		"deathTimer":    player.DeathTimer,
		"totalDeaths":   player.Deaths,
		"jailTimer":     player.JailTimer,
		"oob":           player.OOB,
		"sentence":      player.Sentence,
		"spree":         player.Spree,
		"status":        player.Status,
	}).Debug("Dead")
}

// getHeistResults returns the results of the heist, which contains the outcome
// for each member of the heist crew.
func getHeistResults(server *Server, target *Target) *HeistResult {
	log.Trace("--> getHeistResults")
	defer log.Trace("<-- getHeistResults")

	results := &HeistResult{
		target: target,
	}
	results.memberResults = make([]*HeistMemberResult, 0, len(server.Heist.Crew))
	results.survivingCrew = make([]*HeistMemberResult, 0, len(server.Heist.Crew))

	theme := themes[server.Config.Theme]
	goodResults := theme.Good
	badResults := theme.Bad
	successRate := calculateSuccessRate(server.Heist, target)

	for _, playerID := range server.Heist.Crew {
		player := server.Players[playerID]
		chance := rand.Intn(100) + 1
		log.WithFields(log.Fields{"Player": player.Name, "Chance": chance, "SuccessRate": successRate}).Debug("Heist Results")
		if chance <= successRate {
			index := rand.Intn(len(goodResults))
			goodResult := goodResults[index]
			updatedResults := make([]GoodMessage, 0, len(goodResults))
			updatedResults = append(updatedResults, goodResults[:index]...)
			goodResults = append(updatedResults, goodResults[index+1:]...)
			if len(goodResults) == 0 {
				goodResults = theme.Good
			}

			result := &HeistMemberResult{
				player:       player,
				status:       FREE,
				message:      goodResult.Message,
				bonusCredits: goodResult.Amount,
			}
			results.memberResults = append(results.memberResults, result)
			results.survivingCrew = append(results.survivingCrew, result)
			results.escaped++
		} else {
			index := rand.Intn(len(badResults))
			badResult := badResults[index]
			updatedResults := make([]BadMessage, 0, len(badResults))
			updatedResults = append(updatedResults, badResults[:index]...)
			badResults = append(updatedResults, badResults[index+1:]...)
			if len(badResults) == 0 {
				badResults = theme.Bad
			}

			result := &HeistMemberResult{
				player:       player,
				status:       badResult.Result,
				message:      badResult.Message,
				bonusCredits: 0,
			}
			results.memberResults = append(results.memberResults, result)
			if result.status == DEAD {
				results.dead++
			} else {
				results.survivingCrew = append(results.survivingCrew, result)
				results.apprehended++
			}
		}
	}

	// If at least one member escaped, then calculate the credits to distributed.
	// Also, if no one member escaped, then set the surviving crew to nil so the
	// "No one made it out alive" message is sent.
	log.WithFields(log.Fields{"Escaped": results.escaped, "Apprehended": results.apprehended, "Dead": results.dead}).Debug("Heist Results")
	if results.escaped > 0 {
		calculateCredits(results)
	} else {
		results.survivingCrew = nil
	}

	return results
}

// getTarget returns the target with the smallest maximum crew size that exceeds the number of
// crew members. If no target matches the criteria, then the target with the maximum crew size
// is used.
func getTarget(heist *Heist, targets map[string]*Target) *Target {
	log.Trace("--> getTarget")
	defer log.Trace("<-- getTarget")

	crewSize := int64(len(heist.Crew))
	var target *Target
	for _, possible := range targets {
		if possible.CrewSize >= crewSize {
			if target == nil || target.CrewSize > possible.CrewSize {
				target = possible
			}
		}
	}
	log.WithField("Target", target.ID).Debug("Heist Target")
	return target
}

// vaultUpdater updates the vault periodically so each vault will, over time, recover its credits after being
// hit by a raid.
func vaultUpdater() {
	const timer = time.Duration(1 * time.Minute)
	time.Sleep(20 * time.Second)
	for {
		for _, server := range servers {
			save := false
			for _, target := range server.Targets {
				vault := hmath.Min(target.Vault+(target.VaultMax*4/100), target.VaultMax)
				if vault != target.Vault {
					log.WithFields(log.Fields{"Target": target.ID, "Old": target.Vault, "New": vault, "Max": target.VaultMax}).Debug("Updating Vault")
					target.Vault = vault
					save = true
				}
			}
			if save {
				store.Store.Save(HEIST, server.ID, server)
			}
			save = false
		}
		time.Sleep(timer)
	}
}

// GetMemberHelp returns help information about the heist bot commands for regular members.
func GetMemberHelp() []string {
	help := make([]string, 0, len(playerCommands[0].Options))

	for _, subcommand := range playerCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/heist %s**:  %s\n", subcommand.Name, subcommand.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Heist**\n"}, help...)

	return help
}

// GetAdminHelp returns help information about the heist bot for administrators.
func GetAdminHelp() []string {
	help := make([]string, 0, len(adminCommands[0].Options))

	for _, command := range adminCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/heist-admin %s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Heist**\n"}, help...)

	return help
}
