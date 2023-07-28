package heist

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/rbrabson/heist/pkg/economy"
	log "github.com/sirupsen/logrus"
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

// policeAlert returns how much time is remaining for the cooldown phase after a heist.
func policeAlert(server *Server) time.Duration {
	if server.Config.AlertTime.IsZero() {
		return 0
	}
	if time.Now().After(server.Config.AlertTime) {
		server.Config.AlertTime = time.Time{}
		return 0
	}
	timeRemaining := time.Until(server.Config.AlertTime)
	return timeRemaining
}

// heistChecks returns an error, with appropriate message, if a heist cannot be started.
func heistChecks(server *Server, player *Player, targets map[string]*Target) (string, bool) {
	theme := themes[server.Config.Theme]
	bank := economy.GetBank(banks, server.ID)

	if len(targets) == 0 {
		msg := "Oh no! There are no targets!"
		return msg, false
	}
	if server.Heist != nil && contains(server.Heist.Crew, player.ID) {
		msg := fmt.Sprintf("You are already in the %s.", theme.Crew)
		return msg, false
	}
	if player.Status == "Apprehended" {
		if player.JailTimer.After(time.Now()) && !player.OOB {
			bailCost := server.Config.BailBase
			if player.OOB {
				bailCost *= 3
			}
			remainingTime := time.Until(player.JailTimer)
			msg := fmt.Sprintf("You are in %s. You are serving a %s of %s.\nYou can wait out your remaining %s of: %s, or pay %d credits to be relased on %s.",
				theme.Jail, theme.Sentence, fmtDuration(player.Sentence), theme.Sentence, fmtDuration(remainingTime), bailCost, theme.Bail)
			return msg, false
		}

		msg := fmt.Sprintf("Looks like your %s is over, but you're still in %s! Get released released by typing `/heist release`.", theme.Sentence, theme.Jail)
		return msg, false
	}
	if player.Status == "Dead" {
		fmt.Println("DeathTimer:", player.DeathTimer, ", Now:", time.Now())
		if player.DeathTimer.After(time.Now()) {
			remainingTime := time.Until(player.DeathTimer)
			msg := fmt.Sprintf("You are dead. You will revive in %s", fmtDuration(remainingTime))
			return msg, false
		}
		msg := "Looks like you are still dead, but you can revive at anytime by using the command `/heist revive`."
		return msg, false
	}
	account := bank.GetAccount(player.ID, player.Name)
	if account.Balance < int(server.Config.HeistCost) {
		msg := fmt.Sprintf("You do not have enough credits to cover the cost of entry. You need %d credits to participate", server.Config.HeistCost)
		return msg, false
	}

	alertTime := policeAlert(server)
	if alertTime != 0 {
		msg := fmt.Sprintf("The %s are on high alert after the last target. We should wait for things to cool off before hitting another target. Time remaining: %s.", theme.Police, fmtDuration(alertTime))
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
func handleHeistFailure(server *Server, player *Player, badResult BadMessage) {
	if badResult.Result == "Apprehended" {
		sentence := server.Config.SentenceBase * player.JailCounter
		bail := server.Config.BailBase
		if player.OOB {
			bail *= 3
		}
		player.Status = "Apprehended"
		player.BailCost = bail
		player.Sentence = time.Duration(sentence)
		player.JailTimer = time.Now()
		player.JailCounter += 1
		player.TotalJail += 1
		player.CriminalLevel += 1

		return
	}

	player.CriminalLevel = 0
	player.OOB = false
	player.BailCost = 0
	player.Sentence = 0
	player.Status = "Dead"
	player.JailCounter = 0
	player.DeathTimer = time.Now().Add(server.Config.DeathTimer)
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

			result := &HeistMemberResult{
				player:       player,
				status:       "free",
				message:      goodResult.Message,
				bonusCredits: goodResult.Amount,
			}
			results.memberResults = append(results.memberResults, result)
			results.survivingCrew = append(results.survivingCrew, result)
		} else {
			index := rand.Intn(len(goodResults))
			badResult := badResults[index]
			updatedResults := make([]BadMessage, 0, len(badResults))
			updatedResults = append(updatedResults, badResults[:index]...)
			badResults = append(updatedResults, badResults[index+1:]...)

			result := &HeistMemberResult{
				player:       player,
				status:       badResult.Result,
				message:      badResult.Message,
				bonusCredits: 0,
			}
			results.memberResults = append(results.memberResults, result)
			if result.status != "Dead" {
				results.survivingCrew = append(results.survivingCrew, result)
			}

			handleHeistFailure(server, player, badResult)
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