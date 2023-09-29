package race

import (
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/rbrabson/heist/pkg/store"
)

const (
	MODE = "mode"
)

var (
	Modes map[string]*Mode
)

// Mode represents the type of characters and symbols used in the race.
type Mode struct {
	ID         string       `json:"_id" bson:"_id"`               // Name of the mode (e.g., "clash")
	Beginning  string       `json:"beginning" bson:"beginning"`   // The character or icon for the left-hand side of the race
	Characters []*Character `json:"characters" bson:"characters"` // The list of characters from which a racer is randomly assigned
	Ending     string       `json:"ending" bson:"ending"`         // The character or icon for the right-hand side of the race
}

// Character is one of the characters that may be included in a race.
type Character struct {
	Emoji    string `json:"emoji" bson:"emoji"`       // The icon for the race character
	Movement string `json:"movement" bson:"movement"` // The movement for the race character
}

// GetModeNames returns a list of available race modes.
func GetModeNames(modes map[string]*Mode) ([]string, error) {
	var fileNames []string
	for _, mode := range modes {
		fileNames = append(fileNames, mode.ID)
	}

	return fileNames, nil
}

// LoadModes loads the race modes.
func LoadModes() map[string]*Mode {
	modes := make(map[string]*Mode)
	modeIDs := store.Store.ListDocuments(MODE)
	for _, modeID := range modeIDs {
		var mode Mode
		store.Store.Load(MODE, modeID, &mode)
		modes[mode.ID] = &mode
	}

	return modes
}

// Getode gets the specified race mode.
func GetMode(modeName string) (*Mode, error) {
	theme, ok := Modes[modeName]
	if !ok {
		msg := "Race mode " + modeName + " does not exist."
		log.Warning(msg)
		return nil, fmt.Errorf(msg)
	}

	return theme, nil
}

// String returns a string representation of the race mode.
func (m *Mode) String() string {
	out, _ := json.Marshal(*m)
	return string(out)
}
