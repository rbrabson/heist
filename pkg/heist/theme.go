package heist

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	themeDir string
)

// init loads in the directory that contains the theme files.
func init() {
	godotenv.Load()
	themeDir = os.Getenv("HEIST_FILE_THEME_DIR")
}

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
func LoadThemes(store Store) map[string]*Theme {
	themes := store.LoadThemes()

	return themes
}

// LoadTheme gets the specified theme and returns.
func LoadTheme(themeName string) (*Theme, error) {
	fileName := themeDir + themeName + ".json"
	file, err := os.ReadFile(fileName)
	if err != nil {
		log.Warning("Unable to load theme, error:", err)
		return nil, fmt.Errorf("unable to load theme `%s`", themeName)
	}

	var theme Theme
	if err = json.Unmarshal(file, &theme); err != nil {
		log.Warning("Unable to parse "+fileName, ".json, error:", err)
		return nil, fmt.Errorf("invalid theme format for `%s`", themeName)
	}
	return &theme, nil
}

// String returns a string representation of the theme.
func (t *Theme) String() string {
	out, _ := json.Marshal(*t)
	return string(out)
}
