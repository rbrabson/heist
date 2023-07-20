package economy

import "time"

// Economy contains all the credits for members on different servers.
type Economy struct {
	ID           string                  `json:"_id" bson:"_id"`
	Global       Global                  `json:"global" bson:"global"`
	Guilds       map[string]Guild        `json:"guilds" bson:"guilds"`
	GuildMembers map[string]GuildMembers `json:"guild_members" bson:"guild_members"`
}

type Global struct {
	SchemaVersion int `json:"schema_version"`
}

type Guild struct {
	ID             string `json:"_id" bson:"_id"`
	BankName       string `json:"bank_name" bson:"bank_name"`
	Currency       string `json:"currency" bson:"currency"`
	DefaultBalance int    `json:"default_balance" bson:"default_balance"`
}

type GuildMembers struct {
	ID      string            `json:"_id" bson:"_id"`
	Members map[string]Member `json:"members" bson:"members"`
}
type Member struct {
	ID        string    `json:"_id" bson:"_id"`
	Balance   int       `json:"balance" bson:"balance"`
	CreatedAt time.Time `json:"created_at" bson:"created_at"`
	Name      string    `json:"name" bson:"name"`
}
