package admin

import (
	"fmt"
	"strings"
)

type config struct {
	adminEmails map[string]struct{}
	adminIDs    map[string]struct{}
}

type ConfigOption func(*config)

func WithAdminEmails(emails ...string) ConfigOption {
	return func(c *config) {
		if c.adminEmails == nil {
			c.adminEmails = make(map[string]struct{}, len(emails))
		}
		for _, email := range emails {
			email = strings.ToLower(strings.TrimSpace(email))
			if email != "" {
				c.adminEmails[email] = struct{}{}
			}
		}
	}
}

func WithAdminUserIDs(ids ...any) ConfigOption {
	return func(c *config) {
		if c.adminIDs == nil {
			c.adminIDs = make(map[string]struct{}, len(ids))
		}
		for _, id := range ids {
			c.adminIDs[fmt.Sprint(id)] = struct{}{}
		}
	}
}
