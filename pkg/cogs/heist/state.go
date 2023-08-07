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

// Server contains the data for a given server with the specific ID.
type Server struct {
	ID      string             `json:"_id" bson:"_id"`
	Config  Config             `json:"config" bson:"config"`
	Players map[string]*Player `json:"players" bson:"players"`
	Targets map[string]*Target `json:"targets" bson:"targets"`
	Heist   *Heist             `json:"-" bson:"-"`
}

// Config is the configuration data for a given server.
type Config struct {
	AlertTime    time.Time     `json:"alert_time" bson:"alert_time"`
	BailBase     int64         `json:"bail_base" bson:"bail_base"`
	CrewOutput   string        `json:"crew_output" bson:"crew_output"`
	DeathTimer   time.Duration `json:"death_timer" bson:"death_timer"`
	Hardcore     bool          `json:"hardcore" bson:"hardcore"`
	HeistCost    int64         `json:"heist_cost" bson:"heist_cost"`
	PoliceAlert  time.Duration `json:"police_alert" bson:"police_alert"`
	SentenceBase time.Duration `json:"sentence_base" bson:"sentence_base"`
	Theme        string        `json:"theme" bson:"theme"`
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
	StartTime     time.Time                    `json:"start_time" bson:"start_time"`
	Interaction   *discordgo.InteractionCreate `json:"-" bson:"-"`
	Timer         *waitTimer                   `json:"-" bson:"-"`
}

// Player is a specific player of the heist game on a given server.
type Player struct {
	ID            string        `json:"_id" bson:"_id"`
	BailCost      int64         `json:"bail_cost" bson:"bail_cost"`
	CriminalLevel CriminalLevel `json:"criminal_level" bson:"criminal_level"`
	DeathTimer    time.Time     `json:"death_timer" bson:"death_timer"`
	Deaths        int64         `json:"deaths" bson:"deaths"`
	JailCounter   int64         `json:"jail_counter" bson:"jail"`
	Name          string        `json:"name" bson:"name"`
	OOB           bool          `json:"oob" bson:"oob"`
	Sentence      time.Duration `json:"sentence" bson:"sentence"`
	Spree         int64         `json:"spree" bson:"spree"`
	Status        string        `json:"status" bson:"status"`
	JailTimer     time.Time     `json:"time_served" bson:"time_served"`
	TotalJail     int64         `json:"total_jail" bson:"total_jail"`
}

// Target is a target of a heist.
type Target struct {
	ID       string  `json:"_id" bson:"_id"`
	CrewSize int64   `json:"crew" bson:"crew"`
	Success  float64 `json:"success" bson:"success"`
	Vault    int64   `json:"vault" bson:"vault"`
	VaultMax int64   `json:"vault_max" bson:"vault_max"`
}

// NewServer creates a new server with the specified ID. It is typically called when
// the first call from a server is made to the heist bot.
func NewServer(guildID string) *Server {
	defaultTheme := os.Getenv("HEIST_DEFAULT_THEME")
	if defaultTheme == "" {
		log.Fatal("Default theme not set in environment variable `HEIST_DEFAULT_THEME`")
	}
	theme, err := GetTheme(defaultTheme)
	if err != nil {
		log.Fatal("Unable to load the default theme, error:", err)
	}
	log.Debug(theme)

	server := Server{
		ID: guildID,
		Config: Config{
			AlertTime:    time.Time{},
			BailBase:     250,
			CrewOutput:   "None",
			DeathTimer:   45,
			Hardcore:     false,
			HeistCost:    1500,
			PoliceAlert:  60,
			SentenceBase: 5,
			Theme:        defaultTheme,
			WaitTime:     time.Duration(60 * time.Second),
		},
		Players: make(map[string]*Player, 1),
		Targets: make(map[string]*Target, 1),
	}
	return &server
}

// NewPlayer creates a new player. It is typically called when a player
// first plans or joins a heist.
func NewPlayer(id string, username string, nickname string) *Player {
	player := Player{
		ID:     id,
		Status: FREE,
	}
	if nickname != "" {
		player.Name = nickname
	} else {
		player.Name = username
	}
	return &player
}

// NewHeist creates a new default heist.
func NewHeist(server *Server, planner *Player) *Heist {
	heist := Heist{
		Planner:       planner.ID,
		Crew:          make([]string, 0, 5),
		SurvivingCrew: make([]string, 0, 5),
		StartTime:     time.Now().Add(server.Config.WaitTime),
	}
	heist.Crew = append(heist.Crew, heist.Planner)

	return &heist
}

// NewTarget creates a new target for a heist
func NewTarget(id string, maxCrewSize int64, success float64, vaultCurrent int64, maxVault int64) *Target {
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
func GetServer(servers map[string]*Server, guildID string) *Server {
	server := servers[guildID]
	if server == nil {
		server = NewServer(guildID)
		servers[server.ID] = server
	}
	return server
}

// LoadServers loads all the heist servers from the store.
func LoadServers() map[string]*Server {
	servers := make(map[string]*Server)
	serverIDs := store.Store.ListDocuments(HEIST)
	for _, serverID := range serverIDs {
		var server Server
		store.Store.Load(HEIST, serverID, &server)
		servers[server.ID] = &server
	}

	return servers
}

// GetPlayer returns the player on the server. If the player does not already exist, one is created.
func (s *Server) GetPlayer(id string, username string, nickname string) *Player {
	player, ok := s.Players[id]
	if !ok {
		player = NewPlayer(id, username, nickname)
		s.Players[player.ID] = player
	} else {
		if nickname != "" {
			player.Name = nickname
		} else {
			player.Name = username
		}
	}

	return player
}

// String returns a string representation of the criminal level.
func (cl CriminalLevel) String() string {
	if cl >= Immortal {
		return "Immortal"
	}
	if cl >= Legend {
		return "Legend"
	}
	if cl >= WarChief {
		return "War Chief"
	}
	if cl >= Commander {
		return "Commander"
	}
	if cl >= Veteran {
		return "Veteran"
	}
	if cl >= Renegade {
		return "Renegade"
	}
	return "Greenhorn"
}

// RemainingJailTime returns the amount of time remaining on the player's sentence has been served.
func (p *Player) RemainingJailTime() time.Duration {
	if p.JailTimer.IsZero() || p.JailTimer.After(time.Now()) {
		return 0
	}
	return time.Until(p.JailTimer)
}

// RemainingDeathTime returns the amount of time before the player can be resurected.
func (p *Player) RemainingDeathTime() time.Duration {
	if p.DeathTimer.IsZero() || p.DeathTimer.After(time.Now()) {
		return 0
	}
	return time.Until(p.DeathTimer)
}

// ClearJailAndDeathStatus removes the jail and death times. This is used if the player
// is no longer in jail or has been revived.
func (p *Player) ClearJailAndDeathStatus() {
	p.Status = FREE
	p.DeathTimer = time.Time{}
	p.BailCost = 0
	p.Sentence = 0
	p.JailTimer = time.Time{}
	p.OOB = false
}

// Reset clears the jain and death settings for a player.
func (p *Player) Reset() {
	p.Status = FREE
	p.CriminalLevel = Greenhorn
	p.JailCounter = 0
	p.DeathTimer = time.Time{}
	p.BailCost = 0
	p.Sentence = 0
	p.JailTimer = time.Time{}
	p.OOB = false
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
