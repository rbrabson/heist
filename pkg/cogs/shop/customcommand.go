package shop

// Allow users to purchase a custom command

type Command struct {
	Item
	UserToNotify string `json:"user_to_notify" bson:"user_to_notify"`
}
