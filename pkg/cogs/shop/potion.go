package shop

// Allow users to purchase potions, and figure out how to manage them

type Potion struct {
	Item
	EffectPercent int `json:"effect_percent,omitempty" bson:"effect_percent,omitempty"`
}

func PurchasePotion() {

}
