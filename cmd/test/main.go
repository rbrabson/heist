package main

import (
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/cogs/race"
	"github.com/rbrabson/heist/pkg/math"
	log "github.com/sirupsen/logrus"
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
	numRacers := 11
	r := &race.Race{
		Racers: make([]*race.Racer, 0, 10),
	}
	for i := 0; i < numRacers; i++ {
		index := strconv.Itoa(i)
		player := &race.Player{ID: index, Name: index}
		racer := &race.Racer{Player: player}
		r.Racers = append(r.Racers, racer)
	}

	getRacerButtons(r)

	/*
		godotenv.Load()

		token := os.Getenv("BOT_TOKEN")
		s, err := discordgo.New("Bot " + token)
		if err != nil {
			log.Fatal("Failed to create new bot, error:", err)
		}

		bot := &Bot{
			Session: s,
			timer:   make(chan int),
		}
		bot.Session.Identify.Intents = botIntents

		err = bot.Session.Open()
		if err != nil {
			log.Fatal(err)
		}
		defer bot.Session.Close()

		channelID := "1133474546121449492"

		embeds := []*discordgo.MessageEmbed{
			{
				Type:  discordgo.EmbedTypeRich,
				Title: "Monthly Leaderboard",
				Fields: []*discordgo.MessageEmbedField{
					{
						Value: "This is where the data would go",
					},
				},
			},
		}
		_, err = bot.Session.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Embeds: embeds,
		})
		if err != nil {
			log.Fatal("Unable to send montly leaderboard, err:", err)
		}
	*/

	/*
		lastSeason := time.Date(2023, time.September, 1, 0, 0, 0, 0, time.UTC)
		fmt.Printf("%s %d Leaderboard\n", lastSeason.Month().String(), lastSeason.Year())
	*/
}
