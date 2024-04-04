package race

import (
	"fmt"
	"math/rand"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rbrabson/heist/pkg/math"

	"github.com/rbrabson/heist/pkg/store"
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

// NOTES:
// - 2 second delay between each racer's progress

const (
	RACE = "race"
)

var (
	Servers  map[string]*Server
	Track    = strings.Repeat("•   ", 20)
	TrackLen = int64(utf8.RuneCountInString(Track))
)

// Server represents a guild/server where the Race game is played
type Server struct {
	ID            string             `json:"_id" bson:"_id"`                         // Guild ID
	Config        *Config            `json:"config" bson:"config"`                   // Server-specific configuration
	GamesPlayed   int                `json:"games_played" bson:"games_played"`       // Number of race games played on the server
	Players       map[string]*Player `json:"players" bson:"players"`                 // All members who have entered a race on the server
	LastRaceEnded time.Time          `json:"last_race_ended" bson:"last_race_ended"` // Time the last race ended
	Race          *Race              `json:"-" bson:"-"`                             // The current race (don't save to the store)
	mutex         sync.Mutex         `json:"-" bson:"-"`                             // Lock for updating the server
}

// Config is the race configuration for a given guild/server.
type Config struct {
	BetAmount        int           `json:"bet_amount" bson:"bet_amount"`                 // The amount a player bets on the race
	Currency         string        `json:"currency" bson:"currency"`                     // Currency type used on the server
	Mode             string        `json:"mode" bson:"mode"`                             // The name of the race mode being used
	PrizeMin         int           `json:"prize_min" bson:"prize_min"`                   // The minimum prize for winning racer, multiplied by the number of racers
	PrizeMax         int           `json:"prize_max" bson:"prize_max"`                   // The maximum prize for the winning racer, multiplied by the numbe of racers
	WaitForJoin      time.Duration `json:"wait_for_join" bson:"wait_for_join"`           // Time to wait for people to join a race
	WaitForBetting   time.Duration `json:"wati_for_betting" bson:"wait_for_betting"`     // Time to wait for people to place bets
	WaitBetweenRaces time.Duration `json:"wait_between_races" bson:"wait_between_races"` // Time to wait between races
	MinRacers        int           `json:"min_racers" bson:"min_racers"`                 // Minimum number of racers required, including the bot
	MaxRacers        int           `json:"max_racers" bson:"max_racers"`                 // Maximum number of racers allowed, including the bot
}

// Player is a member of the guild/server who partipates in races or bets on races.
type Player struct {
	ID       string          `json:"_id" bson:"_id"`             // ID of the player
	Name     string          `json:"name" bson:"name"`           // Nickname of the user, or username if the member doesn't have a nickname
	NumRaces int             `json:"num_races" bson:"num_races"` // Number of races the member has entered
	Results  LifetimeResults `json:"results" bson:"results"`     // Results of all previous races for the member
}

// LifetimeResults keeps track of the lifetime results for a given player.
type LifetimeResults struct {
	Win         int `json:"win" bson:"win"`                   // Number of races a player came in first
	Place       int `json:"place" bson:"place"`               // Number of races a player came in second
	Show        int `json:"show" bson:"show"`                 // Number of races a player came in third
	Losses      int `json:"loses" bson:"loses"`               // Number of races a player came in fourth or lower
	Earnings    int `json:"earnings" bson:"earnings"`         // Lifetime earnings for the player in races
	BetsPlaced  int `json:"bets_placed" bson:"bets_placed"`   // Number of bets a player has placed on races
	BetsWon     int `json:"bets_won" bson:"bets_won"`         // Number of bets a player has won
	BetEarnings int `json:"bet_earnings" bson:"bet_earnings"` // Lifetime earnings from placing bets
}

// Race represents the race data for a given guild/server.
type Race struct {
	Bets        []*Bettor                    `json:"bets" bson:"bets"`                 // Bets placed by members on the race
	Racers      []*Racer                     `json:"racers" bson:"racers"`             // Members who have entered the race
	Planned     bool                         `json:"planned" bson:"planned"`           // Waiting for members to join the race
	Started     bool                         `json:"started" bson:"started"`           // The race is now in progress
	Ended       bool                         `json:"ended" bson:"ended"`               // The race has ended
	StartTime   time.Time                    `json:"start_time" bson:"start_time"`     // Time the race will begin
	BetEndTime  time.Time                    `json:"bet_end_time" bson:"bet_end_time"` // The time betting will close
	Interaction *discordgo.InteractionCreate `json:"-" bson:"-"`                       // Interaction for the initial race start message
}

// Racer is a player who is entered into a race.
type Racer struct {
	Player       *Player    `json:"player" bson:"player"`               // Player who is racing
	Character    *Character `json:"character" bson:"character"`         // Randomly selected character for the race
	LastMove     int64      `json:"last_move" bson:"last_move"`         // Distance moved on the last move
	LastPosition int64      `json:"last_position" bson:"last_position"` // Initialize to TrackLen
	Position     int64      `json:"position" bson:"position"`           // Initialize to TrackLen
	Speed        float64    `json:"speed" bson:"speed"`                 // Calculate at end to sort the racers
	Current      string     `json:"current" bson:"current"`             // Current position on the track
	Turn         int64      `json:"turn" bson:"turn"`                   // How many turns it took to move from the starting position to 0
	Prize        int        `json:"prize" bson:"prize"`                 // The amount of credits earned in the race
}

// Bettor is a member who has placed a bet on the outcome of the race.
type Bettor struct {
	ID       string `json:"_id" bson:"_id"`           // Member ID of the player who placed the bet
	Name     string `json:"name" bson:"name"`         // The nickname or username of the player who placed the bet
	Racer    *Racer `json:"racer" bson:"racer"`       // The racer the bet was placed on
	Bet      int    `json:"bet" bson:"bet"`           // The amount of the bet placed
	Winnings int    `json:"winnings" bson:"winnings"` // The amount won on the race
}

// NewServer creates a new server with the default values set and stores it in the file store.
func NewServer(guildID string) *Server {
	log.Trace("--> NewServer")
	defer log.Trace("<-- NewServer")

	server := &Server{
		ID:      guildID,
		Config:  NewConfig(),
		Players: make(map[string]*Player),
	}
	return server
}

// GetServer gets a server for the given guild ID, creating a new one if necessary.
func GetServer(guildID string) *Server {
	log.Trace("--> GetServer")
	defer log.Trace("<-- GetServer")

	server, ok := Servers[guildID]
	if !ok {
		server = NewServer(guildID)
		Servers[server.ID] = server
	}
	return server
}

// NewConfig creates a configuration for a new server. The configuration will use the default values, which
// may be overwritten using commands sent to the bot.
func NewConfig() *Config {
	log.Trace("--> NewConfig")
	defer log.Trace("<-- NewConfig")

	modeName := "clash"
	mode, ok := Modes[modeName]
	if !ok {
		log.Errorf("Unable to load characters for mode %s", modeName)
		return nil
	}
	config := &Config{
		BetAmount:        100,
		Currency:         "credit",
		Mode:             mode.ID,
		PrizeMin:         750,
		PrizeMax:         1250,
		MinRacers:        2,
		MaxRacers:        10,
		WaitForJoin:      time.Duration(30 * time.Second),
		WaitForBetting:   time.Duration(30 * time.Second),
		WaitBetweenRaces: time.Duration(1 * time.Minute),
	}
	return config
}

// NewPlayer creates a new race game player for the server.
func NewPlayer(server *Server, playerID string, playerName string) *Player {
	log.Trace("--> NewPlayer")
	defer log.Trace("<-- NewPlayer")

	player := &Player{
		ID:   playerID,
		Name: playerName,
	}
	return player
}

// GetPlayer returns the race game player, creating the player if one does not already exist.
func (server *Server) GetPlayer(playerID string, username string, nickname string) *Player {
	log.Trace("--> GetPlayer")
	defer log.Trace("<-- GetPlayer")

	var playerName string
	if nickname != "" {
		playerName = nickname
	} else {
		playerName = username
	}
	player, ok := server.Players[playerID]
	if !ok {
		player = NewPlayer(server, playerID, playerName)
		server.Players[playerID] = player
	} else {
		player.Name = playerName
	}
	return player
}

// NewRace creates a new race to be run between multiple racers
func NewRace(server *Server) *Race {
	log.Trace("--> NewRace")
	defer log.Trace("<-- NewRace")

	race := &Race{
		Bets:       make([]*Bettor, 0, 5),
		Racers:     make([]*Racer, 0, server.Config.MaxRacers),
		Planned:    true,
		StartTime:  time.Now().Add(server.Config.WaitForJoin),
		BetEndTime: time.Now().Add(server.Config.WaitForJoin + server.Config.WaitForBetting),
	}
	return race
}

// NewRacer gets a new racer for the given player
func NewRacer(player *Player, mode *Mode) *Racer {
	log.Trace("--> NewRacer")
	defer log.Trace("<-- NewRacer")

	index := rand.Intn(len(mode.Characters))
	character := mode.Characters[index]
	current := fmt.Sprintf("%s %s", Track, character.Emoji)
	racer := &Racer{
		Player:       player,
		Position:     TrackLen,
		LastPosition: TrackLen,
		Character:    character,
		Current:      current,
	}

	return racer
}

// getRacer takes a custom button ID and returns the corresponding racer.
func (race *Race) getRacer(customID string) *Racer {
	log.Trace("--> getRacer")
	defer log.Trace("<-- getRacer")

	switch customID {
	case racers[0]:
		return race.Racers[0]
	case racers[1]:
		return race.Racers[1]
	case racers[2]:
		return race.Racers[2]
	case racers[3]:
		return race.Racers[3]
	case racers[4]:
		return race.Racers[4]
	case racers[5]:
		return race.Racers[5]
	case racers[6]:
		return race.Racers[6]
	case racers[7]:
		return race.Racers[7]
	case racers[8]:
		return race.Racers[8]
	case racers[9]:
		return race.Racers[9]
	case racers[10]:
		return race.Racers[10]
	}

	log.Errorf("Invalid custom ID: %s", customID)
	return nil
}

// calculateRacerWinnings sets the winnings for each of the racers
func calculateRacerWinnings(server *Server) {
	log.Trace("--> calculateRacerWinnings")
	defer log.Trace("<-- calculateRacerWinnings")

	racers := server.Race.Racers
	prize := rand.Intn(int(server.Config.PrizeMax-server.Config.PrizeMin)) + server.Config.PrizeMin
	prize *= len(racers)
	racers[0].Prize = prize
	racers[1].Prize = int((float64(prize) * 0.75))
	if len(racers) > 2 {
		racers[2].Prize = int((float64(prize) * 0.5))
	}
}

// calcualteBetWinnings sets the winnings for each of the bettors
func calcualteBetWinnings(server *Server) {
	log.Trace("--> calcualteBetWinnings")
	defer log.Trace("<-- calcualteBetWinnings")

	winningBet := server.Config.BetAmount * len(server.Race.Racers)
	winner := server.Race.Racers[0]
	for _, bet := range server.Race.Bets {
		if bet.Racer == winner {
			bet.Winnings = winningBet
		}
	}
}

// getCurrentTrack returns the current position of all racers on the track
func getCurrentTrack(racers []*Racer, mode *Mode) string {
	log.Trace("--> getCurrentTrack")
	defer log.Trace("<-- getCurrentTrack")

	var track strings.Builder
	for _, racer := range racers {
		line := fmt.Sprintf("%s **%s %s** [%s]\n", mode.Ending, racer.Current, mode.Beginning, racer.Player.Name)
		track.WriteString(line)
	}
	return track.String()
}

// runLeg runs a single leg of a race - that is, one turn for each player to move.
// It returns the updated track layout as well as a boolean that indicates whether
// the race is over (true) or ongoing (false).
func runLeg(racers []*Racer) bool {
	log.Trace("--> runLeg")
	defer log.Trace("<-- runLeg")

	done := true
	for _, racer := range racers {
		racerDone := racer.Move()
		if racerDone {
			done = false
		}
	}
	return done
}

// RunRace runs the race until all racers have finished.
func (s *Server) RunRace(channelID string) {
	log.Trace("--> RunRace")
	defer log.Trace("<-- RunRace")

	mode := Modes[s.Config.Mode]
	racers := s.Race.Racers
	track := getCurrentTrack(racers, mode)
	message, err := session.ChannelMessageSend(channelID, fmt.Sprintf("%s\n", track))
	if err != nil {
		log.Error("Failed to send message at the start of the race, error:", err)
	}
	messageID := message.ID
	time.Sleep(1 * time.Second)

	done := false
	for !done {
		time.Sleep(2 * time.Second)
		done = runLeg(racers)
		track = getCurrentTrack(racers, mode)
		_, err = session.ChannelMessageEdit(channelID, messageID, fmt.Sprintf("%s\n", track))
		if err != nil {
			log.Error("Failed to update race message, error:", err)
		}
	}
	sort.Slice(racers, func(i, j int) bool {
		if racers[i].Speed == racers[j].Speed {
			return rand.Intn(2) == 0
		}
		return racers[i].Speed < racers[j].Speed
	})
}

// calculateMovement calculates the distance a racer moves on a given turn
func (r *Racer) calculateMovement() int {
	log.Trace("--> calculateMovement")
	defer log.Trace("<-- calculateMovement")

	c := r.Character
	switch c.Movement {
	case "veryfast":
		return rand.Intn(8) * 2
	case "fast":
		return rand.Intn(5) * 3
	case "slow":
		return (rand.Intn(3) + 1) * 3
	case "steady":
		return 2 * 3
	case "abberant":
		chance := rand.Intn(100)
		if chance > 90 {
			return 5 * 3
		}
		return rand.Intn(3) * 3
	case "predator":
		if r.Turn%2 == 0 {
			return 0
		} else {
			return (rand.Intn(4) + 2) * 3
		}
	case "special":
		fallthrough
	default:
		switch r.Turn {
		case 0:
			return 14 * 3
		case 1:
			return 0
		default:
			return rand.Intn(3) * 3
		}
	}
}

// Move moves a player on the track. It returns `true` if the player moved,
// `false` if the player has already finished the race.
func (r *Racer) Move() bool {
	log.Trace("--> Move")
	defer log.Trace("<-- Move")

	if r.Position > 0 {
		r.LastPosition = r.Position
		r.LastMove = int64(r.calculateMovement())
		r.Position -= r.LastMove
		pos := math.Max(0, r.Position)
		start, end := splitString(Track, int(pos))
		r.Current = start + r.Character.Emoji + end
		r.Turn++
		if r.Position <= 0 {
			r.Speed = float64(r.Turn) + float64(r.LastPosition)/float64(r.LastMove)
		}
		return true
	}

	return false
}

// LoadServers returns all the servers for the given guilds.
func LoadServers() {
	log.Trace("--> LoadServers")
	defer log.Trace("<-- LoadServers")

	Servers = make(map[string]*Server)
	serverIDs := store.Store.ListDocuments(RACE)
	for _, serverID := range serverIDs {
		var server Server
		store.Store.Load(RACE, serverID, &server)
		Servers[server.ID] = &server
	}
}

// SaveServer saves the race statistics for the server.
func SaveServer(server *Server) {
	log.Trace("--> SaveServer")
	defer log.Trace("<-- SaveServer")

	store.Store.Save(RACE, server.ID, server)
}

// GetMemberHelp returns help information about the race game commands for regular members.
func GetMemberHelp() []string {
	help := make([]string, 0, len(playerCommands[0].Options))

	for _, subcommand := range playerCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/race %s**:  %s\n", subcommand.Name, subcommand.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Race**\n"}, help...)

	return help
}

// GetAdminHelp returns help information about the race game for administrators.
func GetAdminHelp() []string {
	help := make([]string, 0, len(adminCommands[0].Options))

	for _, command := range adminCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/race-admin %s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Race**\n"}, help...)

	return help
}

// Start initializes anything needed by the race game.
func Start(s *discordgo.Session) {
	session = s
	Modes = LoadModes()
	LoadServers()
}
