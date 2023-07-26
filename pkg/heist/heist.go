package heist

import (
	"math/rand"
	"time"

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
		player.TimeServed = time.Now()
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

	if server.Config.Hardcore {
		// TODO: zero out the player's bank account
	}
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
