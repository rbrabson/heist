package heist

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

type CriminalLevel int

const (
	Greenhorn CriminalLevel = 0
	Renegade  CriminalLevel = 1
	Veteran   CriminalLevel = 10
	Commander CriminalLevel = 25
	WarChief  CriminalLevel = 50
	Legend    CriminalLevel = 75
	Immortal  CriminalLevel = 100
)

// Servers maps the ID of each server to the server settings.
type Servers struct {
	ID      string             `json:"id" bson:"_id"`
	Servers map[string]*Server `json:"servers"`
}

// Server contains the data for a given server with the specific ID.
type Server struct {
	ID      string             `json:"id" bson:"_id"`
	Config  Config             `json:"config" bson:"config"`
	Players map[string]*Player `json:"players" bson:"players"`
	Targets map[string]*Target `json:"targets" bson:"targets"`
	Theme   Theme              `json:"theme" bson:"theme"`
	Heist   *Heist             `json:"-" bson:"-"`
}

// Config is the configuration data for a given server.
// TODO: should the timer values be time.Duration instead of ints?
type Config struct {
	AlertTime    int           `json:"alert_time" bson:"alert_time"`
	BailBase     int           `json:"bail_base" bson:"bail_base"`
	CrewOutput   string        `json:"crew_output" bson:"crew_output"`
	DeathTimer   int           `json:"death_timer" bson:"death_timer"`
	Hardcore     bool          `json:"hardcore" bson:"hardcore"`
	HeistCost    int           `json:"heist_cost" bson:"heist_cost"`
	PoliceAlert  int           `json:"police_alert" bson:"police_alert"`
	SentenceBase int           `json:"sentence_base" bson:"sentence_base"`
	Theme        string        `json:"theme" bson:"theme"`
	Version      string        `json:"version" bson:"version"`
	WaitTime     time.Duration `json:"wait_time" bson:"wait_time"`
}

// Heist is the data for a heist that is either planned or being executed.
type Heist struct {
	Planner       string                       `json:"planner" bson:"planner"`
	Crew          []string                     `json:"crew" bson:"crew"`
	SurvivingCrew []string                     `json:"surviving_crew" bson:"surviving_crew"`
	Planned       bool                         `json:"planned" bson:"planned"`
	Started       bool                         `json:"started" bson:"started"`
	MessageID     string                       `json:"message_id" bson:"message_id"`
	Interaction   *discordgo.InteractionCreate `json:"-" bson:"-"`
	Timer         *waitTimer
}

// Player is a specific player of the heist game on a given server.
type Player struct {
	ID            string        `json:"id" bson:"_id"`
	BailCost      int           `json:"bail_cost" bson:"bail_cost"`
	CriminalLevel CriminalLevel `json:"criminal_level" bson:"criminal_level"`
	DeathTimer    int           `json:"death_timer" bson:"death_timer"`
	Deaths        int           `json:"deaths" bson:"deaths"`
	JailCounter   int           `json:"jail_counter" bson:"jail"`
	Name          string        `json:"name" bson:"name"`
	OOB           bool          `json:"oob" bson:"oob"`
	Sentence      int           `json:"sentence" bson:"sentence"`
	Spree         int           `json:"spree" bson:"spree"`
	Status        string        `json:"status" bson:"status"`
	TimeServed    int           `json:"time_served" bson:"time_served"`
	TotalJail     int           `json:"total_jail" bson:"total_jail"`
}

// Target is a target of a heist.
type Target struct {
	ID       string `json:"id" bson:"_id"`
	CrewSize int    `json:"crew_size" bson:"crew_size"`
	Success  int    `json:"success" bson:"success"`
	Vault    int    `json:"vault" bson:"vault"`
	VaultMax int    `json:"vault_max" bson:"vault_max"`
}

// NewServers creates a new set of servers. This is typically called when the heist
// bot is being started.
func NewServers() *Servers {
	state := Servers{
		ID:      "heist",
		Servers: make(map[string]*Server, 0),
	}

	return &state
}

// NewServer creates a new server with the specified ID. It is typically called when
// the first call from a server is made to the heist bot.
func NewServer(guildID string) *Server {
	defaultTheme := os.Getenv("HEIST_DEFAULT_THEME")
	if defaultTheme == "" {
		log.Fatal("default theme not set in environment variable `HEIST_DEFAULT_THEME`")
	}
	theme, err := LoadTheme(defaultTheme)
	if err != nil {
		log.Fatal(err)
	}
	log.Debug(theme)

	server := Server{
		ID: guildID,
		Config: Config{
			AlertTime:    0,
			BailBase:     250,
			CrewOutput:   "None",
			DeathTimer:   45,
			Hardcore:     false,
			HeistCost:    1500,
			PoliceAlert:  60,
			SentenceBase: 5,
			Theme:        defaultTheme,
			Version:      "1.0.0",
			WaitTime:     time.Duration(60 * time.Second),
		},
		Players: make(map[string]*Player, 1),
		Targets: make(map[string]*Target, 1),
		Theme:   *theme,
	}
	return &server
}

// NewPlayer creates a new player. It is typically called when a player
// first plans or joins a heist.
func NewPlayer(playerID string, playerName string) *Player {
	player := Player{
		ID:     playerID,
		Name:   playerName,
		Status: "free",
	}
	return &player
}

// NewHeist creates a new default heist.
func NewHeist(planner *Player) *Heist {
	heist := Heist{
		Planner:       planner.ID,
		Crew:          make([]string, 0, 5),
		SurvivingCrew: make([]string, 0, 5),
	}
	heist.Crew = append(heist.Crew, heist.Planner)

	return &heist
}

// NewTarget creates a new target for a heist
func NewTarget(id string, maxCrewSize int, success int, vaultCurrent int, maxVault int) *Target {
	target := Target{
		ID:       id,
		CrewSize: maxCrewSize,
		Success:  success,
		Vault:    vaultCurrent,
		VaultMax: maxVault,
	}
	return &target

}

// GetServer returns the server for the guild. If the server does not already exist, one is created.
func (servers *Servers) GetServer(guildID string) *Server {
	server := servers.Servers[guildID]
	if server == nil {
		server = NewServer(guildID)
		servers.Servers[server.ID] = server
	}
	return server
}

func StoreServers(store Store, servers *Servers) {
	store.SaveHeistState(servers)
}

func LoadServers(store Store) *Servers {
	servers := store.LoadHeistState()
	return servers

}

// GetPlayer returns the player on the server. If the player does not already exist, one is created.
func (s *Server) GetPlayer(id string, userName string) *Player {
	player, ok := s.Players[id]
	if !ok {
		player = NewPlayer(id, userName)
		s.Players[player.ID] = player
	} else {
		player.Name = userName
	}
	return player
}

// IsPoliceAlerted returns an indication as to whether a new heist can be
// started and, if not, how long before the heist can be started.
func (c *Config) IsPoliceAlerted() (int, bool) {
	if c.AlertTime == 0 {
		return 0, false
	}
	if c.AlertTime- /* time.perf-counter() */ 0 >= c.PoliceAlert {
		return 0, false
	}

	seconds := c.AlertTime - c.PoliceAlert
	log.Info("seconds:", seconds)

	return seconds, true
}

// String returns a string representation of the criminal level.
func (cl CriminalLevel) String() string {
	switch cl {
	case Greenhorn:
		return "Greenhorn"
	case Renegade:
		return "Renegade"
	case Veteran:
		return "Veteran"
	case Commander:
		return "Commander"
	case WarChief:
		return "War Chief"
	case Legend:
		return "Legend"
	case Immortal:
		return "Immortal"
	}
	return "Unknown"
}

// ClearSettings clears the jain and death settings for a player.
func (p *Player) ClearSettings() {
	p.Status = "free"
	p.CriminalLevel = Greenhorn
	p.JailCounter = 0
	p.DeathTimer = 0
	p.BailCost = 0
	p.Sentence = 0
	p.TimeServed = 0
	p.OOB = false
}

// String returns a string representation of all servers on the system.
func (s *Servers) String() string {
	out, _ := json.Marshal(s)
	return string(out)
}

// String returns a string representation of the server.
func (s *Server) String() string {
	out, _ := json.Marshal(s)
	return string(out)
}

// String returns a string representation of the server configuration.
func (c *Config) String() string {
	out, _ := json.Marshal(c)
	return string(out)
}

// String returns a string representation of the player.
func (p *Player) String() string {
	out, _ := json.Marshal(p)
	return string(out)
}

// String returns a string representation of the target.
func (t *Target) String() string {
	out, _ := json.Marshal(t)
	return string(out)
}
