package economy

import "time"

// Economy contains all the credits for members on different servers.
type Economy struct {
	ID           string                  `json:"_id"`
	Global       Global                  `json:"global"`
	Guilds       map[string]Guild        `json:"guilds"`
	GuildMembers map[string]GuildMembers `json:"guild_members"`
}

type Global struct {
	SchemaVersion int `json:"schema_version"`
}

type Guild struct {
	ID             string `json:"_id"`
	BankName       string `json:"bank_name"`
	Currency       string `json:"currency"`
	DefaultBalance int    `json:"default_balance"`
}

type GuildMembers struct {
	ID      string            `json:"_id"`
	Members map[string]Member `json:"members"`
}
type Member struct {
	ID        string    `json:"_id"`
	Balance   int       `json:"balance"`
	CreatedAt time.Time `json:"created_at"`
	Name      string    `json:"name"`
}
