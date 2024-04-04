package main

import (
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/race"
	"github.com/rbrabson/heist/pkg/cogs/shop"
)

const (
/*
botIntents = discordgo.IntentGuilds |

	discordgo.IntentGuildMessages |
	discordgo.IntentDirectMessages |
	discordgo.IntentGuildEmojis
*/
)

var (
	racers = []string{
		"race_bet_one",
		"race_bet_two",
		"race_bet_three",
		"race_bet_four",
		"race_bet_five",
		"race_bet_six",
		"race_bet_seven",
		"race_bet_eight",
		"race_bet_nine",
		"race_bet_ten",
		"race_bet_eleven",
	}
)

type Bot struct {
	Session *discordgo.Session
	// timer   chan int
}

// getRacerButtons returns action rows for the buttons used to vote on the racers.
func getRacerButtons(race *race.Race) []discordgo.ActionsRow {
	log.Trace("--> getRacerButtons")
	defer log.Trace("<-- getRacerButtons")

	buttonsPerRow := 5
	rows := make([]discordgo.ActionsRow, 0, len(race.Racers)/buttonsPerRow)

	log.Println("")
	log.Println("Starting getRacerButtons")
	// TODO: check all the math here before restarting....
	racersIncludedInButtons := 0
	for len(race.Racers) > racersIncludedInButtons {
		racersNotInButtons := len(race.Racers) - racersIncludedInButtons
		buttonsForNextRow := math.Min(buttonsPerRow, racersNotInButtons)
		buttons := make([]discordgo.MessageComponent, 0, buttonsForNextRow)
		for j := 0; j < buttonsForNextRow; j++ {
			index := j + racersIncludedInButtons
			button := discordgo.Button{
				Label:    race.Racers[index].Player.Name,
				Style:    discordgo.PrimaryButton,
				CustomID: racers[index],
			}
			buttons = append(buttons, button)
		}
		racersIncludedInButtons += buttonsForNextRow

		row := discordgo.ActionsRow{Components: buttons}
		rows = append(rows, row)
		log.WithFields(log.Fields{
			"numRacers": len(race.Racers),
			"buttons":   len(buttons),
			"row":       len(rows),
		}).Println("getRacerButtons")
	}
	log.Println("Ending getRacerButtons")
	log.Println("")

	return rows
}

func main() {
	data := Data{Duration: shop.Duration(DURATION_THREE_DAYS)}

	bytes, _ := json.Marshal(data)
	fmt.Println(string(bytes))

	json.Unmarshal(bytes, &data)
	fmt.Println(time.Duration(data.Duration))

	t, bytes, err := data.Duration.MarshalBSONValue()
	fmt.Println(t, string(bytes), err)

	err = (&data.Duration).UnmarshalBSONValue(t, bytes)
	fmt.Println(data.Duration, err)

	/*
		data, err := os.ReadFile("/Users/roybrabson/dev/heist/cmd/test/test.json")
		if err != nil {
			panic(err)
		}

		var test Test
		if err := json.Unmarshal(data, &test); err != nil {
			panic(err)
		}
		for _, item := range test.Items {
			// var xItem Item
			myItem := item.(map[string]interface{})
			switch myItem["name"] {
			case "string":
				fmt.Println("string")
			case "int":
				fmt.Println("int")
			case "float":
				fmt.Println("float")
			}
		}
	*/

	/*
		var test Test
		err = json.Unmarshal(raw[0], &test.Name)
		if err != nil {
			panic(err)
		}
		for i := range test.Items {
			switch test.Items[i].Name {
			case "string":
				var stringItem StringItem
				if err := json.Unmarshal(raw[i+1], &stringItem); err != nil {
					panic(err)
				}
				// Unmarshal item into a StringItem
			case "int":
				// Unmarhsal item into an IntItem
			case "float":
				// Unmarshal item into a FloatItem
			}
		}
	*/
}
