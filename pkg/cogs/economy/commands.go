package economy

import "github.com/bwmarrin/discordgo"

// Start intializes the economy.
func Start(session *discordgo.Session) {
	LoadBanks()
}

// GetCommands returns the component handlers, command handlers, and commands for the payday bot.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	return nil, nil, nil
}
