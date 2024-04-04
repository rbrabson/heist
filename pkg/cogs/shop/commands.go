package shop

import (
	"fmt"
	"sort"

	"github.com/bwmarrin/discordgo"
	log "github.com/sirupsen/logrus"
)

// TODO: Need an administrative command to withdraw credits from a player's account. This
// should only affect the `Current` balance, and would be used to purchase a custom command.

var (
	commandHandlers = map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"shop":       shop,
		"shop-admin": admin,
	}

	playerCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "shop",
			Description: "Shop commands",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "buy",
					Description: "Buy an item from the shop",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "command",
							Description: "Buy a custom command from the shop",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "potion",
							Description: "Buy a potion from the shop",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "name",
									Description: "The potion to buy",
									Required:    true,
									Type:        discordgo.ApplicationCommandOptionString,
									Choices: []*discordgo.ApplicationCommandOptionChoice{
										{
											Name:  "Fortune",
											Value: "fortune",
										},
										{
											Name:  "Freedom",
											Value: "freedom",
										},
										{
											Name:  "Luck",
											Value: "luck",
										},
										{
											Name:  "Speed",
											Value: "speed",
										},
									},
								},
							},
						},
						{
							Name:        "role",
							Description: "Buy a role from the shop",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Name:        "name",
									Description: "The role to buy",
									Required:    true,
									Type:        discordgo.ApplicationCommandOptionString,
									Choices: []*discordgo.ApplicationCommandOptionChoice{
										{
											Name:  "Gold",
											Value: "gold",
										},
									},
								},
							},
						},
					},
				},
				{
					Name:        "list",
					Description: "List the items available in the shop",
					Type:        discordgo.ApplicationCommandOptionSubCommand,
				},
			},
		},
	}

	adminCommands = []*discordgo.ApplicationCommand{
		{
			Name:        "shop-admin",
			Description: "Shop admin commands.",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "config",
					Description: "Configures the shop.",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "account",
							Description: "Discord ID to send a DM when purchasing custom commands",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
						{
							Name:        "info",
							Description: "Returns the configuration information for the server.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
						},
					},
				},
				{
					Name:        "return",
					Description: "Return an item to the shop",
					Type:        discordgo.ApplicationCommandOptionSubCommandGroup,
					Options: []*discordgo.ApplicationCommandOption{
						{
							Name:        "command",
							Description: "Returns the custom command to the shop.",
							Type:        discordgo.ApplicationCommandOptionSubCommand,
							Options: []*discordgo.ApplicationCommandOption{
								{
									Type:        discordgo.ApplicationCommandOptionString,
									Name:        "id",
									Description: "The member ID.",
									Required:    true,
								},
							},
						},
					},
				},
			},
		},
	}
)

// shop handles player commands within the shop
func shop(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> shop.shop")
	defer log.Trace("<-- shop.shop")
}

// admin handles admin commands within the shop
func admin(s *discordgo.Session, i *discordgo.InteractionCreate) {
	log.Trace("--> shop.admin")
	defer log.Trace("<-- shop.admin")
}

// Start initializes anything needed by the shop.
func Start(s *discordgo.Session) {
	log.Trace("--> shop.Start")
	defer log.Trace("<-- shop.Start")
}

// GetCommands returns the component handlers, command handlers, and commands for the shop.
func GetCommands() (map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate), []*discordgo.ApplicationCommand) {
	log.Trace("--> shop.GetCommands")
	defer log.Trace("<-- shop.GetCommands")

	commands := make([]*discordgo.ApplicationCommand, 0, len(adminCommands)+len(playerCommands))
	commands = append(commands, adminCommands...)
	commands = append(commands, playerCommands...)
	return nil, commandHandlers, commands
}

// GetMemberHelp returns help for member commands
func GetMemberHelp() []string {
	help := make([]string, 0, 1)

	for _, command := range playerCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/shop %s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Shop**\n"}, help...)

	return help
}

// GetAdminHelp returns help for admin commands
func GetAdminHelp() []string {
	help := make([]string, 0, len(adminCommands[0].Options))

	for _, command := range adminCommands[0].Options {
		commandDescription := fmt.Sprintf("- **/shop-admin %s**:  %s\n", command.Name, command.Description)
		help = append(help, commandDescription)
	}
	sort.Slice(help, func(i, j int) bool {
		return help[i] < help[j]
	})
	help = append([]string{"**Shop**\n"}, help...)

	return help
}
