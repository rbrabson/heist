package heist

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

type channelMute struct {
	channel             *discordgo.Channel
	everyoneID          string
	everyonePermissions discordgo.PermissionOverwrite
	s                   *discordgo.Session
	i                   *discordgo.InteractionCreate
}

func newChannelMute(s *discordgo.Session, i *discordgo.InteractionCreate) *channelMute {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		fmt.Println("Error getting channel, error:", err)
	}

	c := channelMute{
		s:       s,
		i:       i,
		channel: channel,
	}

	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		fmt.Println("Error getting roles, error:", err)
	}
	for _, role := range roles {
		if role.Name == "@everyone" {
			c.everyoneID = role.ID
		}
	}

	for _, p := range channel.PermissionOverwrites {
		if p.ID == c.everyoneID {
			c.everyonePermissions = *p
			break
		}
	}

	return &c
}

func (c *channelMute) muteChannel() {
	log.Println("MuteChannel")

	c.s.ChannelPermissionSet(c.i.ChannelID, c.everyoneID, discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionSendMessages|discordgo.PermissionAddReactions)
}

func (c *channelMute) unmuteChannel() {
	log.Println("UnMuteChannel")

	if c.everyonePermissions.ID == "" {
		c.s.ChannelPermissionDelete(c.i.ChannelID, c.everyoneID)
	} else {
		c.s.ChannelPermissionSet(c.i.ChannelID, c.everyoneID, c.everyonePermissions.Type, c.everyonePermissions.Allow, c.everyonePermissions.Deny)
	}
}
