package heist

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/joho/godotenv"
	log "github.com/sirupsen/logrus"
)

var (
	themeDir string
)

// init loads in the directory that contains the theme files.
func init() {
	godotenv.Load()
	themeDir = os.Getenv("HEIST_THEME_DIR")
}

// Theme is a heist theme.
type Theme struct {
	Good []struct {
		Message string `json:"message"`
		Amount  int    `json:"amount"`
	} `json:"good"`
	Bad []struct {
		Message string `json:"message"`
		Result  string `json:"result"`
	} `json:"bad"`
	Jail     string `json:"jail"`
	OOB      string `json:"oob"`
	Police   string `json:"police"`
	Bail     string `json:"bail"`
	Crew     string `json:"crew"`
	Sentence string `json:"sentence"`
	Heist    string `json:"heist"`
	Vault    string `json:"vault"`
}

// GetThemes returns a list of available themes.
func GetThemes() ([]string, error) {
	files, err := os.ReadDir(themeDir)
	if err != nil {
		log.Info(err)
		return nil, err
	}

	var fileNames []string
	for _, file := range files {
		strs := strings.Split(file.Name(), ".json")
		if len(strs) == 2 {
			fileNames = append(fileNames, strs[0])
		}
	}

	return fileNames, nil
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
