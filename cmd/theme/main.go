package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/rbrabson/heist/pkg/cogs/heist"
	log "github.com/sirupsen/logrus"
)

const (
	oldThemeDir = "/Users/roybrabson/dev/electro-dragon/data/heist/"
	newThemeDir = "/Users/roybrabson/dev/heist/configs/theme/"
)

type OldTheme interface{}

func fixLine(old string) string {
	new := strings.ReplaceAll(old, "{}", "%s")
	return new
}

func convert(oldThemeFilename string) {
	parts := strings.Split(oldThemeFilename, ".")
	theme := heist.Theme{
		ID: parts[0],
	}
	theme.Good = make([]heist.GoodMessage, 0, 30)
	theme.Bad = make([]heist.BadMessage, 0, 30)

	data, err := os.ReadFile(oldThemeDir + oldThemeFilename)
	if err != nil {
		log.Warning("Failed to read the data from file "+oldThemeFilename+", error:", err)
	}

	strData := string(data)
	lines := strings.Split(strData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "|Bad|") {
			strs := strings.Split(line, "|Bad| ")
			badParts := strings.Split(strs[1], ", ")
			badMessage := heist.BadMessage{
				Message: fixLine(badParts[0]),
				Result:  badParts[1],
			}
			theme.Bad = append(theme.Bad, badMessage)
		} else if strings.HasPrefix(line, "|Good|") {
			strs := strings.Split(line, "|Good| ")
			goodParts := strings.Split(strs[1], ", ")
			amount, _ := strconv.Atoi(goodParts[1])
			goodMessage := heist.GoodMessage{
				Message: fixLine(goodParts[0]),
				Amount:  amount,
			}
			theme.Good = append(theme.Good, goodMessage)
		} else if strings.HasPrefix(line, "Jail") {
			strs := strings.Split(line, "Jail = ")
			theme.Jail = strs[1]
		} else if strings.HasPrefix(line, "OOB") {
			strs := strings.Split(line, "OOB = ")
			theme.OOB = strs[1]
		} else if strings.HasPrefix(line, "Police") {
			strs := strings.Split(line, "Police = ")
			theme.Police = strs[1]
		} else if strings.HasPrefix(line, "Crew") {
			strs := strings.Split(line, "Crew = ")
			theme.Crew = strs[1]
		} else if strings.HasPrefix(line, "Sentence") {
			strs := strings.Split(line, "Sentence = ")
			theme.Sentence = strs[1]
		} else if strings.HasPrefix(line, "Heist") {
			strs := strings.Split(line, "Heist = ")
			theme.Heist = strs[1]
		} else if strings.HasPrefix(line, "Vault") {
			strs := strings.Split(line, "Vault = ")
			theme.Vault = strs[1]
		} else if strings.HasPrefix(line, "Bail") {
			strs := strings.Split(line, "Bail = ")
			theme.Bail = strs[1]
		} else if line != "" {
			fmt.Println("Unknown line", line)
		}
	}
	data, err = json.Marshal(theme)
	if err != nil {
		log.Fatal("Unable to unmarshal the new theme, error:", err)
	}
	filename := newThemeDir + theme.ID + ".json"
	err = os.WriteFile(filename, data, 0644)
	if err != nil {
		log.Fatal("Unable to write the new theme file, error:", err)
	}
}

func main() {
	godotenv.Load()
	dirEntries, err := os.ReadDir(oldThemeDir)
	if err != nil {
		log.Fatal("Failed to get the list of old theme files, error:", err)
	}

	for _, dirEntry := range dirEntries {
		convert(dirEntry.Name())
	}
	// list of files that are in the directory with the "old".txt files
	// load them one at a time
	// convert them one at a time
	// write the converted data to a file w/ JSON
}
