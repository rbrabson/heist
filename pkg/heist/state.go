package heist

import (
	"encoding/json"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/rbrabson/heist/pkg/store"
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
	Servers map[string]*Server `json:"servers"`
}

// Server contains the data for a given server with the specific ID.
type Server struct {
	ID      string             `json:"id"`
	Config  Config             `json:"config"`
	Players map[string]*Player `json:"players"`
	Targets map[string]*Target `json:"targets"`
	Theme   Theme              `json:"theme"`
	Heist   *Heist             `json:"heist"`
}

// Config is the configuration data for a given server.
// TODO: should the timer values be time.Duration instead of ints?
type Config struct {
	AlertTime    int           `json:"alert_time"`
	BailBase     int           `json:"bail_base"`
	CrewOutput   string        `json:"crew_output"`
	DeathTimer   int           `json:"death_timer"`
	Hardcore     bool          `json:"hardcore"`
	HeistCost    int           `json:"heist_cost"`
	PoliceAlert  int           `json:"police_alert"`
	SentenceBase int           `json:"sentence_base"`
	Theme        string        `json:"theme"`
	Version      string        `json:"version"`
	WaitTime     time.Duration `json:"wait_time"`
}

// Heist is the data for a heist that is either planned or being executed.
type Heist struct {
	Planner       string                       `json:"planner"`
	Crew          []string                     `json:"crew"`
	SurvivingCrew []string                     `json:"surviving_crew"`
	Planned       bool                         `json:"planned"`
	Started       bool                         `json:"started"`
	MessageID     string                       `json:"message_id"`
	Interaction   *discordgo.InteractionCreate `json:"interaction"`
	Timer         *waitTimer
}

// Player is a specific player of the heist game on a given server.
type Player struct {
	ID            string        `json:"id"`
	BailCost      int           `json:"bail_cost"`
	CriminalLevel CriminalLevel `json:"criminal_level"`
	DeathTimer    int           `json:"death_timer"`
	Deaths        int           `json:"deaths"`
	JailCounter   int           `json:"jail_counter"`
	Name          string        `json:"name"`
	OOB           bool          `json:"oob"`
	Sentence      int           `json:"sentence"`
	Spree         int           `json:"spree"`
	Status        string        `json:"status"`
	TimeServed    int           `json:"time_served"`
	TotalJail     int           `json:"total_jail"`
}

// Target is a target of a heist.
type Target struct {
	ID       string `json:"id"`
	CrewSize int    `json:"crew_size"`
	Success  int    `json:"success"`
	Vault    int    `json:"vault"`
	VaultMax int    `json:"vault_max"`
}

// NewServers creates a new set of servers. This is typically called when the heist
// bot is being started.
func NewServers() *Servers {
	state := Servers{
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

func StoreServers(store store.Store, servers *Servers) {
	data, err := json.MarshalIndent(servers, "", " ")
	if err != nil {
		log.Fatal(err)
	}
	store.SaveHeistState(data)
}

func LoadServers(store store.Store) *Servers {
	data, err := store.LoadHeistState()
	if err != nil {
		log.Info("no server data found, returning new server")
		return NewServers()
	}
	var servers Servers
	err = json.Unmarshal(data, &servers)
	if err != nil {
		log.Error("unable to unmarshal server data")
		return NewServers()
	}
	return &servers

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
