package models

import "time"

type Discorduser struct {
	Uid            int
	Userid         string
	Username       string
	Discriminator  string
	Xp             int
	Nextxpgaintime time.Time
	Xpexcluded     bool
	Reputation     int
	Cangivereptime time.Time
}
