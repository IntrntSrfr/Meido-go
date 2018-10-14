package commands

import (
	"github.com/bwmarrin/discordgo"
)

type command func(s *discordgo.Session, m *discordgo.MessageCreate)

var commands map[string]command = make(map[string]command)

func Register(name string, handlerFunc command) {
	commands[name] = handlerFunc
}
