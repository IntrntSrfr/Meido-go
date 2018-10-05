package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
)

type Config struct {
	Token            string
	DmMessageChannel string
}

var (
	startTime time.Time
)

func main() {

	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("Config file not found.")
		return
	}

	var config Config
	json.Unmarshal(file, &config)

	token := config.Token
	client, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println(err)
		return
	}

	addHandlers(client)

	err = client.Open()

	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	client.Close()
}

func addHandlers(s *discordgo.Session) {
	s.AddHandler(readyHandler)
	//s.AddHandler(presenceUpdatedHandler)
	s.AddHandler(messageReceivedHandler)
}

func readyHandler(s *discordgo.Session, m *discordgo.Ready) {
	startTime = time.Now()
	fmt.Println(s.State.User.Username, "#", s.State.User.Discriminator)
}

func presenceUpdatedHandler(s *discordgo.Session, m *discordgo.PresenceUpdate) {
	if m.Game == nil || m.User.Bot || m.User.ID == "172002275412279296" {
		return
	}

	d, _ := json.MarshalIndent(m, "", "\t")

	//	s.ChannelMessageSend("496570247629897738", fmt.Sprintf("%v\n", string(d)))

	if int(m.Game.Type) == 1 {
		s.ChannelMessageSend("496570247629897738", fmt.Sprintf("%v", string(d)))
	}
}

func messageReceivedHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("Config file not found.")
		return
	}

	var config Config
	json.Unmarshal(file, &config)

	args := strings.Split(m.Content, " ")

	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	ch, err := s.Channel(m.ChannelID)
	if err != nil {
		return
	}

	if ch.Type == discordgo.ChannelTypeDM {
		if strings.ToLower(m.Content) == "enroll me" {
			cfc, err := s.Guild("320896491596283906")
			if err != nil {
				return
			}

			var enrolledRole *discordgo.Role

			groles, err := s.GuildRoles(cfc.ID)
			if err != nil {
				return
			}

			for i := range groles {
				role := groles[i]
				if role.ID == "404333507918430212" {
					enrolledRole = role
				}
			}

			if enrolledRole == nil {
				return
			}

			for i := range cfc.Members {
				member := cfc.Members[i]

				if member.User.ID == m.Author.ID {
					err := s.GuildMemberRoleAdd(cfc.ID, m.Author.ID, enrolledRole.ID)
					if err != nil {
						return
					}
				}
			}
			return
		}
		s.ChannelMessageSend(config.DmMessageChannel, fmt.Sprintf("%v#%v (%v)\n%v - %v", m.Author.Username, m.Author.Discriminator, m.Author.ID, ch.ID, m.Content))
		return
	}

	//fmt.Printf("%v#%v - %v\n", m.Author.Username, m.Author.Discriminator, m.Content)

	if args[0] == "m?ban" || args[0] == ".b" {

		var targetUser *discordgo.User
		var reason string
		var pruneDays int

		g, err := s.Guild(ch.GuildID)
		if err != nil {
			return
		}

		_, err = s.GuildMember(g.ID, m.Author.ID)
		if err != nil {
			return
		}

		perms, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
		if err != nil {
			return
		}

		if perms&discordgo.PermissionBanMembers == 0 {
			return
		}

		if len(args) <= 1 {
			return
		}

		if len(args) >= 3 {
			pruneDays, err = strconv.Atoi(args[2])
			if err != nil {
				pruneDays = 0
				reason = strings.Join(args[2:], " ")
			} else {
				reason = strings.Join(args[3:], " ")
			}
			if pruneDays > 7 {
				pruneDays = 7
			}
		}

		if len(m.Mentions) >= 1 {
			targetUser = m.Mentions[0]
		} else {
			targetUser, err = s.User(args[1])
			if err != nil {
				return
			}
		}

		if targetUser.ID == m.Author.ID {
			s.ChannelMessageSend(ch.ID, "no")
			return
		}
		userchannel, err := s.UserChannelCreate(targetUser.ID)
		if err != nil {
			return
		}

		if reason == "" {
			s.ChannelMessageSend(userchannel.ID, fmt.Sprintf("you just got a sick ban kiddo"))

		} else {
			s.ChannelMessageSend(userchannel.ID, fmt.Sprintf("you just got a sick ban for the following reason: %v", reason))
		}

		err = s.GuildBanCreateWithReason(g.ID, targetUser.ID, fmt.Sprintf("%v#%v - %v", m.Author.Username, m.Author.Discriminator, reason), pruneDays)
		if err != nil {
			s.ChannelMessageSend(ch.ID, err.Error())
			return
		}

		embed := &discordgo.MessageEmbed{
			Title: "🚷 User banned",
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Username",
					Value:  fmt.Sprintf("%v", targetUser.Mention()),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "ID",
					Value:  fmt.Sprintf("%v", targetUser.ID),
					Inline: true,
				},
			},

			Color: 13107200,
		}

		s.ChannelMessageSendEmbed(ch.ID, embed)

	}
	if args[0] == "m?unban" || args[0] == ".unban" {
		if len(args) <= 1 {
			return
		}

		g, err := s.Guild(ch.GuildID)
		if err != nil {
			return
		}

		perms, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
		if err != nil {
			return
		}

		if perms&discordgo.PermissionBanMembers == 0 {
			s.ChannelMessageSend(m.ChannelID, "u cant ban lole")
			return
		}

		userID := args[1]

		err = s.GuildBanDelete(g.ID, userID)
		if err != nil {
			return
		}

		targetUser, err := s.User(userID)
		if err != nil {
			return
		}

		embed := &discordgo.MessageEmbed{
			Description: fmt.Sprintf("**Unbanned** %v - %v#%v (%v)", targetUser.Mention(), targetUser.Username, targetUser.Discriminator, targetUser.ID),
			Color:       51200,
		}

		s.ChannelMessageSendEmbed(ch.ID, embed)
	}

	if args[0] == "m?uptime" {

		thisTime := time.Now()

		timespan := thisTime.Sub(startTime)

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Uptime: %v", timespan.String()))

	}

	if args[0] == "m?withnick" {
		var (
			newName       string
			matchingUsers int
		)

		if len(args) <= 1 {
			return
		}

		newName = strings.Join(args[1:], " ")

		g, err := s.Guild(ch.GuildID)
		if err != nil {
			return
		}

		for i := 0; i < len(g.Members); i++ {
			member := g.Members[i]

			if strings.ToLower(member.Nick) == strings.ToLower(newName) {
				matchingUsers++
			}
		}

		s.ChannelMessageSend(ch.ID, fmt.Sprintf("Users with nickname %v: %v", newName, matchingUsers))
	}

	if m.Author.ID == "163454407999094786" {

		if args[0] == "m?server" {
			server := args[1]

			g, err := s.Guild(server)
			if err != nil {
				s.ChannelMessageSend(m.ChannelID, err.Error())
				return
			}

			d, _ := json.MarshalIndent(g, "", "\t")

			s.ChannelMessageSend(m.ChannelID, string(d))
		}
		if args[0] == "m?user" {
			server := args[1]

			g, err := s.User(server)
			if err != nil {
				return
			}

			d, _ := json.MarshalIndent(g, "", "\t")

			s.ChannelMessageSend(m.ChannelID, string(d))
		}

		if args[0] == "m?listall" {
			g, err := s.Guild(ch.GuildID)
			if err != nil {
				return
			}

			for i := range g.Members {
				member := g.Members[i]

				d, err := json.MarshalIndent(member, "", "\t")
				if err != nil {
					return
				}

				s.ChannelMessageSend(ch.ID, string(d))
			}
		}
		if args[0] == "m?renameall" {
			var (
				newName           string
				successfulRenames int
				failedRenames     int
			)

			g, err := s.Guild(ch.GuildID)
			if err != nil {
				return
			}

			if len(args) >= 2 {
				newName = strings.Join(args[1:], " ")

				if len(newName) > 32 {
					s.ChannelMessageSend(ch.ID, "Keep it at 32 characters or less ty")
					return
				}
			}

			for i := 0; i < len(g.Members); i++ {
				member := g.Members[i]

				member.Nick = newName
				err := s.GuildMemberNickname(g.ID, member.User.ID, newName)
				if err != nil {
					failedRenames++
				} else {
					successfulRenames++
				}
			}

			s.ChannelMessageSend(ch.ID, fmt.Sprintf("Renamed %v member(s). Failed to rename %v member(s)", successfulRenames, failedRenames))

		}
		if args[0] == "m?invert" {
			/*
				ch, err := s.Channel(m.ChannelID)
				if err != nil {
					return
				}
					g, err := s.Guild(ch.GuildID)
					if err != nil {
						return
					}
				var attachment *discordgo.MessageAttachment

				if len(m.Attachments) >= 1 {
					attachment = m.Attachments[0]
				}

				if attachment == nil {
					return
				}

				out, err := os.Create("image.png")
				defer out.Close()

				resp, err := http.Get(attachment.URL)
				defer resp.Body.Close()

				n, err := io.Copy(out, resp.Body)

				file, err := os.Open("image.png")

				ip := image.NewNRGBA(image.Rect(0, 0, attachment.Width, attachment.Height))

				s.ChannelMessageSend(ch.ID, fmt.Sprintf("%v", n))
			*/
		}
		if args[0] == "m?ping" {
			sendTime := time.Now()

			msg, err := s.ChannelMessageSend(ch.ID, "Pong")
			if err != nil {
				return
			}

			receiveTime := time.Now()

			delay := receiveTime.Sub(sendTime)

			s.ChannelMessageEdit(ch.ID, msg.ID, "Pong - "+delay.String())
		}

		if args[0] == "m?dm" {
			user := args[1]

			userch, err := s.UserChannelCreate(user)

			if err != nil {
				s.ChannelMessageSend(m.ChannelID, err.Error())
				return
			}

			s.ChannelMessageSend(userch.ID, strings.Join(args[2:], " "))
		}
		if args[0] == "m?msg" {
			var ch string

			if strings.HasPrefix(args[1], "<#") && strings.HasSuffix(args[1], ">") {
				ch = args[1]
				ch = ch[2 : len(ch)-1]
			} else {
				ch = args[1]
			}

			s.ChannelMessageSend(ch, strings.Join(args[2:], " "))
		}
		if args[0] == "m?lockdown" {
			g, err := s.Guild(ch.GuildID)
			if err != nil {
				s.ChannelMessageSend(ch.ID, "Error getting guild: "+err.Error())
				return
			}

			gr, err := s.GuildRoles(g.ID)
			if err != nil {
				s.ChannelMessageSend(ch.ID, "Error getting guildroles: "+err.Error())
				return
			}

			for i := 0; i < len(gr); i++ {
				/*
					gamer, err := json.Marshal(gr[i])
					if err != nil {
						s.ChannelMessageSend(ch.ID, "Error fixing output: "+err.Error())
						return
					}
				*/
				if gr[i].Name == "@everyone" {
					everyoneRole := gr[i]

					err := s.ChannelPermissionSet(ch.ID, everyoneRole.ID, "role", 0, 2048)
					if err != nil {
						s.ChannelMessageSend(ch.ID, err.Error())
						return
					}

					embed := &discordgo.MessageEmbed{
						Description: "Locked channel.",
						Color:       51200,
					}

					s.ChannelMessageSendEmbed(ch.ID, embed)

					return
				}
				//s.ChannelMessageSend(ch.ID, string(gamer))
			}
		}
		if args[0] == "m?unlock" {

			g, err := s.Guild(ch.GuildID)
			if err != nil {
				s.ChannelMessageSend(ch.ID, "Error getting guild: "+err.Error())
				return
			}

			gr, err := s.GuildRoles(g.ID)
			if err != nil {
				s.ChannelMessageSend(ch.ID, "Error getting guildroles: "+err.Error())
				return
			}

			for i := 0; i < len(gr); i++ {
				if gr[i].Name == "@everyone" {
					everyoneRole := gr[i]

					err := s.ChannelPermissionSet(ch.ID, everyoneRole.ID, "role", 2048, 0)
					if err != nil {
						s.ChannelMessageSend(ch.ID, err.Error())
						return
					}

					embed := &discordgo.MessageEmbed{
						Description: "Unlocked channel.",
						Color:       51200,
					}

					s.ChannelMessageSendEmbed(ch.ID, embed)

					return
				}
			}
		}
		if args[0] == "m?inrole" {
			roleName := strings.Join(args[1:], " ")

			var (
				selectedRole *discordgo.Role
				inroleUsers  string
			)

			g, err := s.Guild(ch.GuildID)
			if err != nil {
				return
			}

			members := g.Members

			guildRoles, err := s.GuildRoles(g.ID)
			if err != nil {
				return
			}

			for i := 0; i < len(guildRoles); i++ {
				if strings.ToLower(guildRoles[i].Name) == strings.ToLower(roleName) {
					selectedRole = guildRoles[i]
				}
			}

			if selectedRole == nil {
				return
			}

			for i := 0; i < len(members); i++ {
				user := members[i]

				for j := 0; j < len(user.Roles); j++ {
					role := user.Roles[j]

					if role == selectedRole.ID {
						inroleUsers += user.User.Username + "#" + user.User.Discriminator + "\n"
					}
				}
			}
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("```\n%v```", inroleUsers))
		}
		if args[0] == "m?mute" {
			var targetUser *discordgo.User
			var err error

			if len(m.Mentions) >= 1 {
				targetUser = m.Mentions[0]
			} else {
				targetUser, err = s.User(args[1])
				if err != nil {
					s.ChannelMessageSend(ch.ID, err.Error())
					return
				}
			}

			g, err := s.Guild(ch.GuildID)
			if err != nil {
				return
			}

			mutedRoles := 0

			for i := 0; i < len(g.Roles); i++ {
				gr := g.Roles[i]

				if gr.Name == "Gamermute" {
					mutedRoles++
				}
			}

			var mutedRole *discordgo.Role

			if mutedRoles <= 0 {
				mutedRole, err = s.GuildRoleCreate(g.ID)
				if err != nil {
					return
				}

				mutedRole, err = s.GuildRoleEdit(g.ID, mutedRole.ID, "Gamermute", mutedRole.Color, mutedRole.Hoist, 0, false)

				for i := 0; i < len(g.Channels); i++ {
					gch := g.Channels[i]
					err := s.ChannelPermissionSet(gch.ID, mutedRole.ID, "role", 0, 2112)
					if err != nil {
						return
					}
				}
			} else {
				for i := 0; i < len(g.Roles); i++ {
					r := g.Roles[i]

					if r.Name == "Gamermute" {
						mutedRole = r
					}
				}
			}

			err = s.GuildMemberRoleAdd(g.ID, targetUser.ID, mutedRole.ID)
			if err != nil {
				s.ChannelMessageSend(ch.ID, err.Error())
				return
			}

		}
		if args[0] == "m?unmute" {

			g, err := s.Guild(ch.GuildID)
			if err != nil {
				return
			}

			var targetUser *discordgo.User

			if len(m.Mentions) >= 1 {
				targetUser = m.Mentions[0]
			} else {
				targetUser, err = s.User(args[1])
				if err != nil {
					s.ChannelMessageSend(ch.ID, err.Error())
					return
				}
			}

			var mutedRole *discordgo.Role

			for i := 0; i < len(g.Roles); i++ {
				role := g.Roles[i]

				if role.Name == "Gamermute" {
					mutedRole = role
				}
			}

			err = s.GuildMemberRoleRemove(g.ID, targetUser.ID, mutedRole.ID)
			if err != nil {
				return
			}
		}
	}
}
