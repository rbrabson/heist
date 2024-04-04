package shop

import (
	"math"
	"time"
)

const (
	MaxDuration       = Duration(time.Duration(math.MaxInt64))
	DurationOneDay    = time.Duration(259200000000000)
	DurationThreeDays = DurationOneDay * 3
)

// TODO: is there any value in having Config? Maybe to populate the shop if it is in a different DB, like a "shop" DB?
type Server struct {
	ID       string             `json:"_id" bson:"_id"`
	Shop     Shop               `json:"shop" bson:"shop"`
	ShopName string             `json:"shop_name" bson:"shop_name"`
	Members  map[string]*Member `json:"members" bson:"members"`
}

// Shop is the list of items available for purchase on the server.
type Shop struct {
	ID       string     `json:"_id" bson:"_id"`
	Commands []*Command `json:"commands" bson:"commands"`
	Potions  []*Potion  `json:"potions" bson:"potions"`
	Roles    []*Role    `json:"roles" bson:"roles"`
}

// Item is the base type for all items in the shop
type Item struct {
	Name        string        `json:"name" bson:"name"`
	Description string        `json:"description" bson:"description"`
	Cost        int           `json:"cost" bson:"cost"`
	Duration    time.Duration `json:"duration" bson:"duration"`
}
