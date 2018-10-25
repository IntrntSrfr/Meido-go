package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/rand"
	"meido-go/models"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	_ "github.com/lib/pq"
)

type Config struct {
	Token            string   `json:"Token"`
	DmMessageChannel []string `json:"DmMessageChannel"`
	Connectionstring string   `json:"Connectionstring"`
}

var (
	startTime  time.Time
	totalUsers int
	db         *sql.DB
	config     Config
)

const (
	dColorRed   = 13107200
	dColorGreen = 51200
)

func main() {

	file, e := ioutil.ReadFile("./config.json")
	if e != nil {
		fmt.Printf("Config file not found.")
		return
	}

	json.Unmarshal(file, &config)

	token := config.Token
	client, err := discordgo.New("Bot " + token)

	if err != nil {
		fmt.Println(err)
		return
	}

	db, err = sql.Open("postgres", config.Connectionstring)
	if err != nil {
		panic("could not connect to db " + err.Error())
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

	defer db.Close()
	client.Close()
}

func addHandlers(s *discordgo.Session) {
	//s.AddHandler(presenceUpdatedHandler)
	go s.AddHandler(guildAvailableHandler)
	go s.AddHandler(guildRoleDeleteHandler)
	go s.AddHandler(messageReceivedHandler)
	go s.AddHandler(readyHandler)
}

func fullHex(hex string) string {
	i := len(hex)

	for i < 6 {
		hex = "0" + hex
		i++
	}

	return hex
}

func random(min, max int) int {
	rand.Seed(time.Now().Unix())
	return rand.Intn(max-min) + min
}

func readyHandler(s *discordgo.Session, m *discordgo.Ready) {
	startTime = time.Now()
	fmt.Println("Logged in as "+s.State.User.Username, "#", s.State.User.Discriminator)
}

func memberJoinedHandler(s *discordgo.Session, m *discordgo.GuildMemberAdd) {
	sqlstr := "INSERT INTO discordusers(userid, username, discriminator, xp, nextxpgaintime, xpexcluded, reputation, cangivereptime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"

	stmt, err := db.Prepare(sqlstr)
	if err != nil {
		return
	}

	if m.User.Bot {
		return
	}

	row := db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", m.User.ID)

	user := models.Discorduser{}

	err = row.Scan(
		&user.Uid,
		&user.Userid,
		&user.Username,
		&user.Discriminator,
		&user.Xp,
		&user.Nextxpgaintime,
		&user.Xpexcluded,
		&user.Reputation,
		&user.Cangivereptime)

	currentTime := time.Now()

	if err != nil {
		if err == sql.ErrNoRows {
			//var lastInsertID int
			_, err := stmt.Exec(m.User.ID, m.User.Username, m.User.Discriminator, 0, currentTime, false, 0, currentTime)
			if err != nil {
				fmt.Println(err)
				return
			}
		}
	}
}

func guildAvailableHandler(s *discordgo.Session, g *discordgo.GuildCreate) {
	sqlstr := "INSERT INTO discordusers(userid, username, discriminator, xp, nextxpgaintime, xpexcluded, reputation, cangivereptime) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"

	stmt, err := db.Prepare(sqlstr)
	if err != nil {
		return
	}

	loadTimeStart := time.Now()

	fmt.Println(g.Name)
	for i := range g.Members {
		m := g.Members[i]

		if m.User.Bot {
			continue
		}

		row := db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", m.User.ID)

		user := models.Discorduser{}

		err = row.Scan(
			&user.Uid,
			&user.Userid,
			&user.Username,
			&user.Discriminator,
			&user.Xp,
			&user.Nextxpgaintime,
			&user.Xpexcluded,
			&user.Reputation,
			&user.Cangivereptime)

		currentTime := time.Now()

		if err != nil {
			if err == sql.ErrNoRows {
				//var lastInsertID int
				_, err := stmt.Exec(m.User.ID, m.User.Username, m.User.Discriminator, 0, currentTime, false, 0, currentTime)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
	}

	loadTimeEnd := time.Now()
	totalLoadTime := loadTimeEnd.Sub(loadTimeStart)

	fmt.Println(fmt.Sprintf("Loaded %v in %v", g.Name, totalLoadTime.String()))
}

func guildRoleDeleteHandler(s *discordgo.Session, m *discordgo.GuildRoleDelete) {
	row := db.QueryRow("SELECT * FROM userroles WHERE guildid=$1 AND roleid=$2", m.GuildID, m.RoleID)

	ur := models.Userrole{}

	err := row.Scan(&ur.Uid,
		&ur.Guildid,
		&ur.Roleid,
		&ur.Userid)
	if err != nil {
		return
	}

	stmt, err := db.Prepare("DELETE FROM userroles WHERE guildid=$1 AND roleid=$2")
	if err != nil {
		return
	}

	_, err = stmt.Exec(m.GuildID, m.RoleID)
	if err != nil {
		return
	}
}

func messageReceivedHandler(s *discordgo.Session, m *discordgo.MessageCreate) {

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
		} else {

			for i := range config.DmMessageChannel {
				dmch := config.DmMessageChannel[i]

				dmembed := discordgo.MessageEmbed{
					Color:       16777215,
					Title:       fmt.Sprintf("Message from %v", m.Author.String()),
					Description: m.Content,
					Footer:      &discordgo.MessageEmbedFooter{Text: m.Author.ID},
					Timestamp:   string(m.Timestamp),
				}

				_, err := s.ChannelMessageSendEmbed(dmch, &dmembed)
				if err != nil {
					fmt.Println(err)
					continue
				}
			}
		}
		return
	}

	perms, err := s.UserChannelPermissions(m.Author.ID, ch.ID)
	if err != nil {
		return
	}

	if perms&discordgo.PermissionManageMessages == 0 {

		rows, _ := db.Query("SELECT phrase FROM filters WHERE guildid = $1", ch.GuildID)

		isIllegal := false

		for rows.Next() {
			filter := models.Filter{}
			err := rows.Scan(&filter.Filter)
			if err != nil {
				continue
			}

			if strings.Contains(m.Content, filter.Filter) {
				isIllegal = true
				break
			}
		}

		if isIllegal {
			s.ChannelMessageDelete(ch.ID, m.ID)
			s.ChannelMessageSend(ch.ID, fmt.Sprintf("%v, you are not allowed to use a banned word/phrase!", m.Author.Mention()))
		}
	}

	row := db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", m.Author.ID)
	if err != nil {
		return
	}

	dbu := models.Discorduser{}

	err = row.Scan(
		&dbu.Uid,
		&dbu.Userid,
		&dbu.Username,
		&dbu.Discriminator,
		&dbu.Xp,
		&dbu.Nextxpgaintime,
		&dbu.Xpexcluded,
		&dbu.Reputation,
		&dbu.Cangivereptime)

	if err != nil {
		return
	}

	currentTime := time.Now()

	diff := dbu.Nextxpgaintime.Sub(currentTime)

	if diff < 0 {
		stmt, err := db.Prepare("UPDATE discordusers SET xp = $1, nextxpgaintime=$2 WHERE userid = $3")
		if err != nil {
			fmt.Println(err)
			return
		}

		randomXp := random(15, 26)

		_, err = stmt.Exec(dbu.Xp+randomXp, currentTime.Add(time.Minute*time.Duration(2)), dbu.Userid)
		if err != nil {
			fmt.Println(err)
			return
		}
	}

	if args[0] == "m?filter" {

		if perms&discordgo.PermissionManageMessages == 0 {
			s.ChannelMessageSend(ch.ID, "You do not have permissions to do that.")
			return
		}

		if len(args) > 1 {

			if len(args) > 2 {

				phrase := strings.Join(args[2:], " ")

				if len(phrase) < 1 {
					return
				}

				if args[1] == "add" {
					row := db.QueryRow("SELECT * FROM filters WHERE guildid=$1 AND phrase=$2;", ch.GuildID, phrase)

					f := models.Filter{}

					err := row.Scan(&f.Uid, &f.Guildid, &f.Filter)
					if err != nil {
						if err != sql.ErrNoRows {
							return
						}

						_, err := db.Exec("INSERT INTO filters (guildid, phrase) VALUES ($1,$2);", ch.GuildID, phrase)
						if err != nil {
							return
						}
						s.ChannelMessageSend(ch.ID, fmt.Sprintf("Added `%v` to the filter.", phrase))
					} else {
						s.ChannelMessageSend(ch.ID, "Phrase is already in the filter.")
					}

				} else if args[1] == "remove" {
					row := db.QueryRow("SELECT * FROM filters WHERE guildid=$1 AND phrase=$2;", ch.GuildID, phrase)

					f := models.Filter{}

					err := row.Scan(&f.Uid, &f.Guildid, &f.Filter)
					if err != nil {
						if err != sql.ErrNoRows {
							return
						}
						fmt.Println(err)
						s.ChannelMessageSend(ch.ID, "Phrase is not in the filter.")
					} else {
						_, err := db.Exec("DELETE FROM filters WHERE guildid=$1 AND phrase=$2;", ch.GuildID, phrase)
						if err != nil {
							return
						}
						s.ChannelMessageSend(ch.ID, fmt.Sprintf("Removed `%v` from the filter.", phrase))
					}
				}
			} else {
				if args[1] == "clear" {

					res, err := db.Exec("DELETE FROM filters WHERE guildid = $1;", ch.GuildID)
					if err != nil {
						return
					}

					affected, err := res.RowsAffected()
					if err != nil {
						return
					}

					if affected == 0 {
						s.ChannelMessageSend(ch.ID, "The filter is empty.")
					} else {
						s.ChannelMessageSend(ch.ID, "Cleared the filter.")
					}
				}
			}
		} else {

			rows, err := db.Query("SELECT * FROM filters WHERE guildid=$1;", ch.GuildID)
			if err != nil {
				if err != sql.ErrNoRows {
					return
				}
				s.ChannelMessageSend(ch.ID, "The filter is empty.")
			}

			filterlist := "```\nList of currently filtered phrases\n"

			for rows.Next() {
				f := models.Filter{}

				err = rows.Scan(&f.Uid, &f.Guildid, &f.Filter)
				if err != nil {
					return
				}

				filterlist += fmt.Sprintf("- %v\n", f.Filter)
			}

			filterlist += "```"

			s.ChannelMessageSend(ch.ID, filterlist)
		}
	}

	if args[0] == "m?avatar" || args[0] == "m?av" || args[0] == ">av" {

		ch, _ := s.Channel(m.ChannelID)

		var targetUser *discordgo.User
		var err error

		if len(args) > 1 {

			if len(m.Mentions) >= 1 {
				targetUser = m.Mentions[0]
			} else {
				targetUser, err = s.User(args[1])
				if err != nil {
					//s.ChannelMessageSend(ch.ID, err.Error())
					return
				}
			}
		}

		if targetUser == nil {
			targetUser = m.Author
		}

		if targetUser.Avatar == "" {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{
				Color:       dColorRed,
				Description: fmt.Sprintf("%v has no avatar set.", targetUser.String()),
			})
		} else {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{
				Color: dColorGreen,
				Title: targetUser.String(),
				Image: &discordgo.MessageEmbedImage{URL: targetUser.AvatarURL("1024")},
			})
		}
	}

	if args[0] == "m?profile" {

		var targetUser *discordgo.User
		if len(args) > 1 {

			if len(m.Mentions) >= 1 {
				targetUser = m.Mentions[0]
			} else {
				targetUser, err = s.User(args[1])
				if err != nil {
					//s.ChannelMessageSend(ch.ID, err.Error())
					return
				}
			}
		}

		if targetUser == nil {
			targetUser = m.Author
		}

		if targetUser.Bot {
			s.ChannelMessageSend(ch.ID, "Bots dont get to join the fun")
			return
		}

		row := db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", targetUser.ID)

		err := row.Scan(
			&dbu.Uid,
			&dbu.Userid,
			&dbu.Username,
			&dbu.Discriminator,
			&dbu.Xp,
			&dbu.Nextxpgaintime,
			&dbu.Xpexcluded,
			&dbu.Reputation,
			&dbu.Cangivereptime)

		if err != nil {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorRed, Description: "User not available"})
			return
		}

		embed := discordgo.MessageEmbed{
			Color:     dColorGreen,
			Title:     fmt.Sprintf("Profile for %v", targetUser.String()),
			Thumbnail: &discordgo.MessageEmbedThumbnail{URL: targetUser.AvatarURL("1024")},
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Experience",
					Value:  strconv.Itoa(dbu.Xp),
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Reputation",
					Value:  strconv.Itoa(dbu.Reputation),
					Inline: true,
				},
			},
		}

		s.ChannelMessageSendEmbed(ch.ID, &embed)
	}

	if args[0] == "m?rep" {

		u := m.Author

		row := db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", u.ID)

		dbu := models.Discorduser{}

		err = row.Scan(
			&dbu.Uid,
			&dbu.Userid,
			&dbu.Username,
			&dbu.Discriminator,
			&dbu.Xp,
			&dbu.Nextxpgaintime,
			&dbu.Xpexcluded,
			&dbu.Reputation,
			&dbu.Cangivereptime)

		if err != nil {
			return
		}

		diff := dbu.Cangivereptime.Sub(currentTime.Add(time.Hour * time.Duration(2)))

		if len(args) < 2 {
			if diff > 0 {
				s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorRed, Description: strings.TrimSuffix(fmt.Sprintf("You can award a reputation point in %v", diff.Round(time.Minute).String()), "0s")})
			} else {
				s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorGreen, Description: "You can award a reputation point."})
			}
			return
		}

		var targetUser *discordgo.User
		if len(m.Mentions) >= 1 {
			targetUser = m.Mentions[0]
		} else {
			targetUser, err = s.User(args[1])
			if err != nil {
				//s.ChannelMessageSend(ch.ID, err.Error())
				return
			}
		}

		if targetUser.Bot {
			s.ChannelMessageSend(ch.ID, "Bots dont get to join the fun")
			return
		}

		if u.ID == targetUser.ID {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorRed, Description: "You cannot award yourself a reputation point."})
			return
		}
		if diff > 0 {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{
				Color: dColorRed,
				//Description: fmt.Sprintf("You can award a reputation point in %dh%dm", int64(diff.Hours()), int64(diff.Round(time.Second)))})
				Description: strings.TrimSuffix(fmt.Sprintf("You can award a reputation point in %v", diff.Round(time.Minute).String()), "0s")})
			return
		}

		row = db.QueryRow("SELECT * FROM discordusers WHERE userid = $1", targetUser.ID)

		dbtu := models.Discorduser{}

		err = row.Scan(
			&dbtu.Uid,
			&dbtu.Userid,
			&dbtu.Username,
			&dbtu.Discriminator,
			&dbtu.Xp,
			&dbtu.Nextxpgaintime,
			&dbtu.Xpexcluded,
			&dbtu.Reputation,
			&dbtu.Cangivereptime)

		if err != nil {
			return
		}

		_, err = db.Exec("UPDATE discordusers SET reputation = $1 WHERE userid = $2", dbtu.Reputation+1, dbtu.Userid)
		if err != nil {
			return
		}
		_, err = db.Exec("UPDATE discordusers SET cangivereptime = $1 WHERE userid = $2", currentTime.Add(time.Hour*time.Duration(24)), dbu.Userid)
		if err != nil {
			return
		}

		s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorGreen, Description: fmt.Sprintf("%v awarded %v a reputation point!", u.Mention(), targetUser.Mention())})
	}

	if args[0] == "m?lb" {

		rows, err := db.Query("SELECT * FROM discordusers WHERE xp > 0 ORDER BY xp DESC LIMIT 10 ")
		if err != nil {
			fmt.Println(err)
			return
		}

		if rows.Err() != nil {
			fmt.Println(rows.Err())
		}

		leaderboard := "```\n"

		place := 1

		for rows.Next() {
			dbu := models.Discorduser{}

			err = rows.Scan(
				&dbu.Uid,
				&dbu.Userid,
				&dbu.Username,
				&dbu.Discriminator,
				&dbu.Xp,
				&dbu.Nextxpgaintime,
				&dbu.Xpexcluded,
				&dbu.Reputation,
				&dbu.Cangivereptime)

			if err != nil {
				fmt.Println(err)
				return
			}

			leaderboard += fmt.Sprintf("#%v - %v#%v - %vxp\n", place, dbu.Username, dbu.Discriminator, dbu.Xp)
			place++
		}
		leaderboard += "```"

		s.ChannelMessageSend(ch.ID, leaderboard)

	}

	if args[0] == "m?rplb" {

		rows, err := db.Query("SELECT * FROM discordusers WHERE reputation > 0 ORDER BY reputation DESC LIMIT 10 ")
		if err != nil {
			fmt.Println(err)
			return
		}

		if rows.Err() != nil {
			fmt.Println(rows.Err())
		}

		leaderboard := "```\n"

		place := 1

		for rows.Next() {
			dbu := models.Discorduser{}

			err = rows.Scan(
				&dbu.Uid,
				&dbu.Userid,
				&dbu.Username,
				&dbu.Discriminator,
				&dbu.Xp,
				&dbu.Nextxpgaintime,
				&dbu.Xpexcluded,
				&dbu.Reputation,
				&dbu.Cangivereptime)

			if err != nil {
				fmt.Println(err)
				return
			}

			leaderboard += fmt.Sprintf("#%v - %v#%v - %v reputation points\n", place, dbu.Username, dbu.Discriminator, dbu.Reputation)
			place++
		}
		leaderboard += "```"

		s.ChannelMessageSend(ch.ID, leaderboard)

	}

	if args[0] == "m?setuserrole" {
		if len(args) < 3 {
			return
		}

		perms, err := s.UserChannelPermissions(m.Author.ID, m.ChannelID)
		if err != nil {
			return
		}

		if perms&discordgo.PermissionManageRoles == 0 {
			s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Color: dColorRed, Description: "You do not have the required permissions."})
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

		if targetUser.Bot {
			s.ChannelMessageSend(ch.ID, "Bots dont get to join the fun")
			return
		}

		g, err := s.Guild(ch.GuildID)
		if err != nil {
			s.ChannelMessageSend(ch.ID, err.Error())
			return
		}

		var selectedRole *discordgo.Role

		for i := range g.Roles {
			role := g.Roles[i]

			if role.ID == args[2] {
				selectedRole = role
			} else if strings.ToLower(role.Name) == strings.ToLower(strings.Join(args[2:], " ")) {
				selectedRole = role
			}
		}

		if selectedRole == nil {
			s.ChannelMessageSend(ch.ID, "Role not found")
			return
		}

		var lastinsertid int
		err = db.QueryRow("INSERT INTO userroles(guildid, userid, roleid) VALUES($1, $2, $3) returning uid", g.ID, targetUser.ID, selectedRole.ID).Scan(&lastinsertid)
		if err != nil {
			s.ChannelMessageSend(ch.ID, err.Error())
			return
		}

		s.ChannelMessageSend(ch.ID, fmt.Sprintf("Bound role **%v** to user **%v#%v**", selectedRole.Name, targetUser.Username, targetUser.Discriminator))

	}

	if args[0] == "m?myrole" {

		if len(args) >= 2 {

			if args[1] == "color" {
				if len(args) != 3 {
					return
				}

				u := m.Author

				g, err := s.Guild(ch.GuildID)
				if err != nil {
					s.ChannelMessageSend(ch.ID, err.Error())
					return
				}

				row := db.QueryRow("SELECT * FROM userroles WHERE guildid=$1 AND userid=$2", g.ID, u.ID)

				ur := models.Userrole{}

				err = row.Scan(&ur.Uid,
					&ur.Guildid,
					&ur.Userid,
					&ur.Roleid)
				if err != nil {
					if err == sql.ErrNoRows {
						s.ChannelMessageSend(ch.ID, "You dont have a custom role set.")
					}
					return
				}

				if strings.HasPrefix(args[2], "#") {
					args[2] = args[2][1:]
				}

				color, err := strconv.ParseInt("0x"+args[2], 0, 64)
				if err != nil {
					s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Description: "Invalid color code.", Color: dColorRed})
					return
				}

				var oldRole *discordgo.Role

				for i := range g.Roles {
					role := g.Roles[i]

					if role.ID == ur.Roleid {
						oldRole = role
						_, err = s.GuildRoleEdit(g.ID, role.ID, role.Name, int(color), role.Hoist, role.Permissions, role.Mentionable)
						if err != nil {
							if strings.Contains(err.Error(), strconv.Itoa(discordgo.ErrCodeMissingPermissions)) {
								s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Description: "Missing permissions.", Color: dColorRed})
								return
							}
							s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Description: "Invalid color code.", Color: dColorRed})
							return
						}
					}
				}

				embed := discordgo.MessageEmbed{
					Color:       int(color),
					Description: fmt.Sprintf("Color changed from #%v to #%v", fullHex(fmt.Sprintf("%X", oldRole.Color)), fullHex(fmt.Sprintf("%X", color))),
				}
				s.ChannelMessageSendEmbed(ch.ID, &embed)
			}

			if args[1] == "name" {

				if len(args) < 3 {
					return
				}

				newName := strings.Join(args[2:], " ")

				u := m.Author

				g, err := s.Guild(ch.GuildID)
				if err != nil {
					s.ChannelMessageSend(ch.ID, err.Error())
					return
				}

				row := db.QueryRow("SELECT * FROM userroles WHERE guildid=$1 AND userid=$2", g.ID, u.ID)

				ur := models.Userrole{}

				err = row.Scan(&ur.Uid,
					&ur.Guildid,
					&ur.Userid,
					&ur.Roleid)
				if err != nil {
					if err == sql.ErrNoRows {
						s.ChannelMessageSend(ch.ID, "You dont have a custom role set.")
					}
					return
				}

				var oldRole *discordgo.Role

				for i := range g.Roles {
					role := g.Roles[i]

					if role.ID == ur.Roleid {
						oldRole = role
						_, err = s.GuildRoleEdit(g.ID, role.ID, newName, role.Color, role.Hoist, role.Permissions, role.Mentionable)
						if err != nil {
							if strings.Contains(err.Error(), strconv.Itoa(discordgo.ErrCodeMissingPermissions)) {
								s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Description: "Missing permissions.", Color: dColorRed})
								return
							}
							s.ChannelMessageSendEmbed(ch.ID, &discordgo.MessageEmbed{Description: "Some error occured: `" + err.Error() + "`.", Color: dColorRed})
							return
						}
					}
				}

				embed := discordgo.MessageEmbed{
					Color:       int(oldRole.Color),
					Description: fmt.Sprintf("Role name changed from %v to %v", oldRole.Name, newName),
				}
				s.ChannelMessageSendEmbed(ch.ID, &embed)
			}
		}
		var targetUser *discordgo.User

		if len(args) > 1 {

			if len(m.Mentions) >= 1 {
				targetUser = m.Mentions[0]
			} else {
				targetUser, err = s.User(args[1])
				if err != nil {
					//s.ChannelMessageSend(ch.ID, err.Error())
					return
				}
			}
		}

		if targetUser == nil {
			targetUser = m.Author
		}

		if targetUser.Bot {
			s.ChannelMessageSend(ch.ID, "Bots dont get to join the fun")
			return
		}

		u := targetUser

		g, err := s.Guild(ch.GuildID)
		if err != nil {
			s.ChannelMessageSend(ch.ID, err.Error())
			return
		}

		row := db.QueryRow("SELECT * FROM userroles WHERE guildid=$1 AND userid=$2", g.ID, u.ID)

		ur := models.Userrole{}

		err = row.Scan(&ur.Uid,
			&ur.Guildid,
			&ur.Userid,
			&ur.Roleid)
		if err != nil {
			if err == sql.ErrNoRows {
				s.ChannelMessageSend(ch.ID, "No custom role set.")
			}
			return
		}

		var customRole *discordgo.Role

		for i := range g.Roles {
			role := g.Roles[i]

			if role.ID == ur.Roleid {
				customRole = role
			}
		}

		embed := discordgo.MessageEmbed{
			Color: int(customRole.Color),
			Title: fmt.Sprintf("Custom role for %v", u.String()),
			Fields: []*discordgo.MessageEmbedField{
				&discordgo.MessageEmbedField{
					Name:   "Name",
					Value:  customRole.Name,
					Inline: true,
				},
				&discordgo.MessageEmbedField{
					Name:   "Color",
					Value:  fmt.Sprintf("#" + fullHex(fmt.Sprintf("%X", customRole.Color))),
					Inline: true,
				},
			},
		}
		s.ChannelMessageSendEmbed(ch.ID, &embed)
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

	if args[0] == "m?ban" || args[0] == ".b" {

		if len(args) <= 1 {
			return
		}

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
			Title: "User banned",
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

			Color: dColorRed,
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
			Color:       dColorGreen,
		}

		s.ChannelMessageSendEmbed(ch.ID, embed)
	}

	if args[0] == "m?uptime" {

		thisTime := time.Now()

		timespan := thisTime.Sub(startTime)

		s.ChannelMessageSend(m.ChannelID, fmt.Sprintf("Uptime: %v", timespan.String()))

	}

	if args[0] == "m?umr" {
		_, err := s.ChannelMessageSend(ch.ID, "https://i.kym-cdn.com/photos/images/original/001/007/574/ec2.jpg")
		if err != nil {
			return
		}
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
		if args[0] == "m!lockdown" {
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

					var everyoneoverwrites *discordgo.PermissionOverwrite

					for i := range ch.PermissionOverwrites {
						perms := ch.PermissionOverwrites[i]
						if perms.ID == everyoneRole.ID {
							everyoneoverwrites = perms
						}
					}

					fmt.Println(everyoneoverwrites.Deny, discordgo.PermissionSendMessages, everyoneoverwrites.Deny&discordgo.PermissionSendMessages)

					if everyoneoverwrites.Deny&discordgo.PermissionSendMessages != 0 {
						return
					}

					err = s.ChannelPermissionSet(ch.ID, everyoneRole.ID, "role", everyoneoverwrites.Allow-discordgo.PermissionSendMessages, everyoneoverwrites.Deny+discordgo.PermissionSendMessages)
					if err != nil {
						s.ChannelMessageSend(ch.ID, err.Error())
						return
					}

					embed := &discordgo.MessageEmbed{
						Description: "Locked channel.",
						Color:       dColorGreen,
					}

					s.ChannelMessageSendEmbed(ch.ID, embed)

					return
				}
				//s.ChannelMessageSend(ch.ID, string(gamer))
			}
		}
		if args[0] == "m!unlock" {

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

					var everyoneoverwrites *discordgo.PermissionOverwrite

					for i := range ch.PermissionOverwrites {
						perms := ch.PermissionOverwrites[i]
						if perms.ID == everyoneRole.ID {
							everyoneoverwrites = perms
						}
					}

					fmt.Println(everyoneoverwrites.Deny, discordgo.PermissionSendMessages, everyoneoverwrites.Deny&discordgo.PermissionSendMessages)

					if everyoneoverwrites.Deny&discordgo.PermissionSendMessages == 0 {
						return
					}

					err := s.ChannelPermissionSet(ch.ID, everyoneRole.ID, "role", everyoneoverwrites.Allow, everyoneoverwrites.Deny-discordgo.PermissionSendMessages)
					if err != nil {
						s.ChannelMessageSend(ch.ID, err.Error())
						return
					}

					embed := &discordgo.MessageEmbed{
						Description: "Unlocked channel.",
						Color:       dColorGreen,
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
