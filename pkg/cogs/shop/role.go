package shop

type Role struct {
	Item
	ID string `json:"id" bson:"id"`
}

func PurchaseRole() {

}

// AssignRole assigns the role to the user that initiated the command
func AssignRole(role string) {
	// Get the server roles, and verify it is available
	// Verify the role isn't already assigned to the user
	// Get the cost of the role
	// Verify the user has enough money, and withdraw it from their account
	// Send message to assign the role
	// If successful
	//    add to the purchased items for the user
	//    save the bank account & shop details
	//    notify the user
	// Else (not successful)
	//    refund the purchase price for the item
	//    notify the user of the failure

	// Find out if the user has the role already. If so, abort.
	// If user doesn't, then find the role in the guild, and use
	//       GuildMemberRoleAdd to add the role to the user
	// Only do this if the user has the money to do so, and make sure the money is
	//       deducted from the user's account, the purchase is added to their shop
	//       account, and everything is successfully saved. If necessary, it can be
	//       recovered if the role assignment were to fail. Or wait for the role
	//       assignment to succeed, then save the money & the purchase and, if it
	//       failed, then remove both.

	// Make sure the bot has the  MANAGE_ROLES permission in settings & make sure it is activated
	// To tell if a user has a role, get the guild member, then check the Roles slice.

	/*
		err := s.GuildMemberRoleAdd(currentGuild, m.Author.ID)
		if err != nil {
		     fmt.Println("Unable to assign role", err)
		}
		if I pass s.GuildRoles(currentGuild.ID)[0], that's the ID?
		🌞 kit 🌚 — 07/22/2017 8:32 PM
		That will get you the first Role in the guild.
		Get the guild from State.
		Then loop through guild.Roles

		for _, role := range session.State.Guild[n].Roles { if role.Name == "Jungle" { return role } }

	*/

}

// GetServerRoles returns the list of roles on the server
func GetServerRoles() {
	// Get a list of roles for the server
}

// GetUserRoles returns the list of roles assigned to the user
func GetUserRoles() {
	// Get a list of roles assigned to the user
}
