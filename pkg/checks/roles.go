package checks

import (
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/bwmarrin/discordgo"
)

func contains(list []string, element string) bool {
	for _, a := range list {
		if strings.EqualFold(a, element) {
			return true
		}
	}
	return false
}

func IsAdmin(roles discordgo.Roles) bool {
	for _, role := range roles {
		if role.Permissions&discordgo.PermissionAdministrator != 0 {
			return true
		}
	}
	return false
}

func IsModerator(roles discordgo.Roles) bool {
	for _, role := range roles {
		if role.Permissions&discordgo.PermissionModerateMembers != 0 {
			return true
		}
	}
	return false
}

func IsServerManager(roles discordgo.Roles) bool {
	for _, role := range roles {
		if role.Permissions&discordgo.PermissionManageServer != 0 {
			return true
		}
	}
	return false
}

func IsAdminOrServerManager(roles discordgo.Roles) bool {
	log.Debug(roles)
	const perm = discordgo.PermissionAdministrator | discordgo.PermissionManageServer
	for _, role := range roles {
		if role.Permissions&perm != 0 {
			return true
		}
	}
	return false
}

func IsAllowed(roles []*discordgo.Role, roleNames []string) bool {
	for _, role := range roles {
		if contains(roleNames, role.Name) {
			return true
		}
	}
	return false
}
