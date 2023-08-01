package heist

import (
	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

// channelMute is used for muting and unmuting a channel on a server
type channelMute struct {
	channel             *discordgo.Channel
	everyoneID          string
	everyonePermissions discordgo.PermissionOverwrite
	s                   *discordgo.Session
	i                   *discordgo.InteractionCreate
}

// newChannelMute creates a channelMute for the given session and interaction.
func newChannelMute(s *discordgo.Session, i *discordgo.InteractionCreate) *channelMute {
	channel, err := s.Channel(i.ChannelID)
	if err != nil {
		log.Error("Error getting channel, error:", err)
	}

	c := channelMute{
		s:       s,
		i:       i,
		channel: channel,
	}

	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		log.Error("Error getting roles, error:", err)
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

// muteChannel sets the channel so that `@everyone`	 can't send messages to the channel.
func (c *channelMute) muteChannel() {
	err := c.s.ChannelPermissionSet(c.i.ChannelID, c.everyoneID, discordgo.PermissionOverwriteTypeRole, 0, discordgo.PermissionSendMessages|discordgo.PermissionAddReactions)
	if err != nil {
		log.Warning("Failed to mute the channel, error:", err)
	}
}

// unmuteChannel resets the permissions for `@everyone` to what they were before the channel was muted.
func (c *channelMute) unmuteChannel() {
	if c.everyonePermissions.ID == "" {
		c.s.ChannelPermissionDelete(c.i.ChannelID, c.everyoneID)
	} else {
		c.s.ChannelPermissionSet(c.i.ChannelID, c.everyoneID, c.everyonePermissions.Type, c.everyonePermissions.Allow, c.everyonePermissions.Deny)
	}
}
