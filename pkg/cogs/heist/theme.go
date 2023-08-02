package heist

import (
	"encoding/json"
	"fmt"

	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"
)

const (
	THEME = "theme"
)

// Theme is a heist theme.
type Theme struct {
	ID       string        `json:"_id" bson:"_id"`
	Good     []GoodMessage `json:"good"`
	Bad      []BadMessage  `json:"bad"`
	Jail     string        `json:"jail" bson:"jail"`
	OOB      string        `json:"oob" bson:"oob"`
	Police   string        `json:"police" bson:"police"`
	Bail     string        `json:"bail" bson:"bail"`
	Crew     string        `json:"crew" bson:"crew"`
	Sentence string        `json:"sentence" bson:"sentence"`
	Heist    string        `json:"heist" bson:"heist"`
	Vault    string        `json:"vault" bson:"vault"`
}

type GoodMessage struct {
	Message string `json:"message" bson:"message"`
	Amount  int    `json:"amount" bson:"amount"`
}

type BadMessage struct {
	Message string `json:"message" bson:"message"`
	Result  string `json:"result" bson:"result"`
}

// GetThemeNames returns a list of available themes.
func GetThemeNames(map[string]*Theme) ([]string, error) {
	var fileNames []string
	for _, theme := range themes {
		fileNames = append(fileNames, theme.ID)
	}

	return fileNames, nil
}

// LoadThemes loads the themes that may be used by the heist bot.
func LoadThemes() map[string]*Theme {
	themes := make(map[string]*Theme)
	themeIDs := store.Store.ListDocuments(THEME)
	for _, themeID := range themeIDs {
		var theme Theme
		store.Store.Load(THEME, themeID, &theme)
		themes[theme.ID] = &theme
	}

	return themes
}

// GetTheme gets the specified theme and returns.
func GetTheme(themeName string) (*Theme, error) {
	theme, ok := themes[themeName]
	if !ok {
		msg := "Theme " + themeName + " does not exist."
		log.Warning(msg)
		return nil, fmt.Errorf(msg)
	}

	return theme, nil
}

// String returns a string representation of the theme.
func (t *Theme) String() string {
	out, _ := json.Marshal(*t)
	return string(out)
}
