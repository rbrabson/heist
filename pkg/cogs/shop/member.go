package shop

import "time"

// Member is a member of a server that has interacted with the shop, such as purchasing an item.
type Member struct {
	ID        string       `json:"_id" bson:"_id"`
	Purchases []*Purchases `json:"purchases" bson:"purchases"`
}

// Purchase is the base type for all purchased items
type Purchase struct {
	PurchasedOn time.Time
}

// PurchasedCommand is a custom command that has been bought by a member of the server
type PurchasedCommand struct {
	Purchase
	Command
}

// PurchasedPotion is a potion that has been bought by a member of the server
type PurchasedPotion struct {
	Purchase
	Potion
}

// PurchasedPotion is a role that has been bought by a member of the server
type PurchasedRole struct {
	Purchase
	Role
}

// Purchases are the purchases made by a member of the server
type Purchases struct {
	Commands []*PurchasedCommand
	Potions  []*PurchasedPotion
	Roles    []*PurchasedRole
}

// listUnexpiredPurchases returns the list of purchases that have not expired
func (m *Member) listUnexpiredPurchases() *Purchases {
	return nil
}

// listExpiredPurchases returns the list of purchases that have expired
func (m *Member) listExpiredPurchases() *Purchases {
	return nil
}

// listAllPurchases returns the list of expired and unexpired purchases
func (m *Member) listAllPurchases() *Purchases {
	return nil
}

// returnCustomCommand removes the custom command from the user's purchase history and refunds
// the purchase amount
func (m *Member) returnCustomCommand(command *Command) {

}

// hasExpired returns an indication as to whether the time since `t` is `>=` to the duration `d`
func hasExpired(t time.Time, d Duration) bool {
	return time.Since(t) >= time.Duration(d)
}
