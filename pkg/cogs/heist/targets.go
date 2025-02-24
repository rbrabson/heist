package heist

import (
	"encoding/json"
	"fmt"

	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
)

const (
	TARGET = "target"
)

// Targets is the set of targets for a given theme
type Targets struct {
	ID      string   `json:"_id" bson:"_id"`
	Targets []Target `json:"targets" bson:"targets"`
}

// Target is a target of a heist.
type Target struct {
	ID       string  `json:"_id" bson:"_id"`
	CrewSize int64   `json:"crew" bson:"crew"`
	Success  float64 `json:"success" bson:"success"`
	Vault    int64   `json:"vault" bson:"vault"`
	VaultMax int64   `json:"vault_max" bson:"vault_max"`
}

// NewTarget creates a new target for a heist
func NewTarget(id string, maxCrewSize int64, success float64, vaultCurrent int64, maxVault int64) *Target {
	target := Target{
		ID:       id,
		CrewSize: maxCrewSize,
		Success:  success,
		Vault:    vaultCurrent,
		VaultMax: maxVault,
	}
	return &target
}

// LoadTargets loads the targets that may be used by the heist bot.
func LoadTargets() map[string]*Targets {
	targetSet := make(map[string]*Targets)
	targetIDs := store.Store.ListDocuments(TARGET)
	for _, targetID := range targetIDs {
		var targets Targets
		store.Store.Load(TARGET, targetID, &targets)
		targetSet[targets.ID] = &targets
	}

	return targetSet
}

// GetTargetSet gets the specified target and returns.
func GetTargetSet(targetName string) (*Targets, error) {
	targets, ok := targetSet[targetName]
	if !ok {
		msg := targetName + " targets do not exist."
		log.Warning(msg)
		return nil, fmt.Errorf("%s", msg)
	}

	return targets, nil
}

// GetTargets gets the specified list of targets and returns.
func GetTargets(targetName string) (*Targets, error) {
	targets, ok := targetSet[targetName]
	if !ok {
		msg := targetName + " targets do not exist."
		log.Warning(msg)
		return nil, fmt.Errorf(msg)
	}

	return targets, nil
}

// String returns a string representation of the targets.
func (t *Targets) String() string {
	out, _ := json.Marshal(t)
	return string(out)
}

// String returns a string representation of the target.
func (t *Target) String() string {
	out, _ := json.Marshal(t)
	return string(out)
}
