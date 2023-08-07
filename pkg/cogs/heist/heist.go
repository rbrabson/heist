package heist

import (
	"fmt"
	"math/rand"
	"sort"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/economy"
	"github.com/rbrabson/heist/pkg/format"
	log "github.com/sirupsen/logrus"
)

const (
	FREE        = "Free"
	DEAD        = "Dead"
	APPREHENDED = "Apprehended"
)

// HeistResult is the result of a heist.
type HeistResult struct {
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
	if server.Heist != nil && contains(server.Heist.Crew, player.ID) {
		msg := fmt.Sprintf("You are already in the %s.", theme.Crew)
		return msg, false
	}
	if player.Status == APPREHENDED && !player.OOB {
		if player.JailTimer.After(time.Now()) {
			remainingTime := time.Until(player.JailTimer)
			msg := fmt.Sprintf("You are in %s. You are serving a %s of %s.\nYou can wait out your remaining %s of: %s, or pay %d credits to be relased on %s.",
				theme.Jail, theme.Sentence, format.Duration(player.Sentence), theme.Sentence, format.Duration(remainingTime), player.BailCost, theme.Bail)
			return msg, false
		}

		msg := p.Sprintf("Looks like your %s is over, but you're still in %s! Get released released by typing `/heist release`.", theme.Sentence, theme.Jail)
		return msg, false
	}
	if player.Status == DEAD {
		if player.DeathTimer.After(time.Now()) {
			remainingTime := time.Until(player.DeathTimer)
			msg := p.Sprintf("You are dead. You will revive in %s", format.Duration(remainingTime))
			return msg, false
		}
		msg := "Looks like you are still dead, but you can revive at anytime by using the command `/heist revive`."
		return msg, false
	}
	account := bank.GetAccount(player.ID, player.Name)
	if account.Balance < int(server.Config.HeistCost) {
		msg := p.Sprintf("You do not have enough credits to cover the cost of entry. You need %d credits to participate", server.Config.HeistCost)
		return msg, false
	}
	if server.Config.AlertTime.After(time.Now()) {
		remainingTime := time.Until(server.Config.AlertTime)
		msg := p.Sprintf("The %s are on high alert after the last target. We should wait for things to cool off before hitting another target. Time remaining: %s.", theme.Police, format.Duration(remainingTime))
		return msg, false
	}

	return "", true
}

// calculateCredits determines the number of credits stolen by each surviving crew member.
func calculateCredits(results *HeistResult) {
	creditsStolenPerSurvivor := int(float64(results.target.Vault) * 0.75 / float64((len(results.memberResults) + len(results.survivingCrew))))
	for _, player := range results.survivingCrew {
		player.stolenCredits = creditsStolenPerSurvivor
	}
}

// calculateBonusRate calculates the bonus amount to add to the success rate
// for a heist. The closer you are to the maximum crew size, the larger
// the bonus amount.
func calculateBonusRate(heist *Heist, target *Target) int {
	log.Debug("--> calculateBonus")
	defer log.Debug("<-- calculateBonus")

	percent := 100 * int64(len(heist.Crew)) / target.CrewSize
	log.WithField("percent", percent).Debug("Bonus Percentage")
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
	log.Debug("--> calculateSuccessRate")
	defer log.Debug("<-- calculateSuccessRate")

	bonus := calculateBonusRate(heist, target)
	successChance := int(target.Success) + bonus
	log.WithField("successRate", successChance).Debug("Success Rate")
	return successChance
}

// handleHeistFailure updates the status of a player who is apprehended or killed during a heist.
func handleHeistFailure(server *Server, player *Player, result *HeistMemberResult) {
	log.Debug("--> handleHeistFailure")
	defer log.Debug("<-- handleHeistFailure")

	if result.status == APPREHENDED {
		sentence := int64(server.Config.SentenceBase) * (player.JailCounter + 1)
		bail := server.Config.BailBase
		if player.OOB {
			bail *= 3
		}
		player.BailCost = bail
		player.CriminalLevel += 1
		player.JailCounter += 1
		player.OOB = false
		player.Sentence = time.Duration(sentence)
		player.JailTimer = time.Now().Add(player.Sentence)
		player.Spree = 0
		player.Status = APPREHENDED
		player.TotalJail += 1

		log.WithFields(log.Fields{
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

	log.WithFields(log.Fields{
		"bail":          player.BailCost,
		"criminalLevel": player.CriminalLevel,
		"deathTimer":    player.DeathTimer,
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
	log.Debug("--> getHeistResults")
	defer log.Debug("<-- getHeistResults")

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
				message:      fmt.Sprintf("%s %s.", badResult.Message, badResult.Result),
				bonusCredits: 0,
			}
			results.memberResults = append(results.memberResults, result)
			if result.status != DEAD {
				results.survivingCrew = append(results.survivingCrew, result)
			}
		}
	}

	calculateCredits(results)

	return results
}

// getTarget returns the target with the smallest maximum crew size that exceeds the number of
// crew members.
func getTarget(heist *Heist, targets map[string]*Target) *Target {
	log.Debug("--> getTarget")
	defer log.Debug("<-- getTarget")

	crewSize := int64(len(heist.Crew))
	var target *Target
	for _, possible := range targets {
		if possible.CrewSize >= crewSize {
			if target == nil || target.CrewSize > possible.CrewSize {
				target = possible
			}
		}
	}
	return target
}

// vaultUpdater updates the vault periodically so each vault will, over time, recover its credits after being
// hit by a raid.
func vaultUpdater() {
	const timer = time.Duration(120 * time.Second)
	time.Sleep(20 * time.Second)
	for {
		for _, server := range servers {
			for _, target := range server.Targets {
				vault := min(target.Vault+(target.VaultMax*4/100), target.VaultMax)
				target.Vault = vault
			}
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
