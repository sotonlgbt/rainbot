package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
)

type UserWarningInformation struct {
	Server, Timeframe                         string
	CurrentVerification, RequiredVerification StudentType
}

var warningText *template.Template

// init loads in the warning text template.
func init() {
	var err error
	warningText, err = template.New("warningText.got").Funcs(template.FuncMap{
		"aOrAn": func(nextWord string) string {
			for _, letter := range []rune{'a', 'e', 'i', 'o', 'u'} {
				if []rune(nextWord)[0] == letter {
					return "an " + nextWord
				}
			}
			return "a " + nextWord
		},
	}).ParseFiles("templates/warningText.got")
	if err != nil {
		log.Fatalln("Failed to parse template file for warning text with err", err)
	}
}

// warnInvalidUsers finds invalid users in a given guild and sends them a warning message,
// notifying them that they may soon be removed for not having verified.
func (b *Bot) warnInvalidUsers(guildID discord.GuildID) {
	guild, err := b.State.Guild(guildID)
	if err != nil {
		log.Fatalln("Failed fetching guild for", guildID, "with error", err)
	}

	memberTypeForGuild := getMemberTypeForGuild(guildID)

	b.findInvalidMembersInGuild(guildID, memberTypeForGuild, func(user discord.User, actualUserType string) {
		log.Println("User", user.Username, "is not correctly authenticated - messaging")
		if !warnInvalidDryRunMode {
			var messageToSend bytes.Buffer
			warningText.Execute(&messageToSend, UserWarningInformation{guild.Name, warnInvalidDeadline, GetStudentTypeFromCode(actualUserType), memberTypeForGuild})

			memberChannel, err := b.State.CreatePrivateChannel(user.ID)
			if err != nil {
				log.Println("Failed creating message channel to user", user.Username, "with error", err)
				return
			}

			b.State.SendMessageComplex(memberChannel.ID, api.SendMessageData{
				Content: messageToSend.String(),
				Components: *discord.ComponentsPtr(
					&discord.ActionRowComponent{
						&discord.ButtonComponent{
							CustomID: discord.ComponentID("verifyme_button_guild_" + guildID.String()),
							Label:    "Let's get verified!",
							Emoji: &discord.ComponentEmoji{
								Name: "🎉",
							},
							Style: discord.PrimaryButtonStyle(),
						},
					},
				),
			})
		}
	})

}

// purgeInvalidUsers finds invalid users in a guild and removes them for not having verified.
func (b *Bot) PurgeInvalidUsers(guildID discord.GuildID) {
	guild, err := b.State.Guild(guildID)
	if err != nil {
		log.Fatalln("Failed fetching guild for", guildID, "with error", err)
	}

	b.findInvalidMembersInGuild(guildID, getMemberTypeForGuild(guildID), func(user discord.User, actualUserType string) {
		log.Println("User", user.Username, "is not correctly authenticated - purging")
		b.State.Kick(guildID, user.ID, api.AuditLogReason("Incorrectly authenticated for this server and an invalid member purge is running - was: "+actualUserType))

		memberChannel, err := b.State.CreatePrivateChannel(user.ID)
		if err != nil {
			log.Println("Failed creating message channel to user", user.Username, "with error", err)
			return
		}

		message, err := b.createReinviteMessage(guildID, user)
		if err != nil {
			log.Fatalln("Failed creating reinvite message for user", user.Username, "with error", err)
		}

		message.Content = fmt.Sprintf(`You weren't verified for the %s server for this academic year, so we've had to say goodbye for now. Need to reverify? Hit the button below.
				No longer the right server for you? There's other opportunities! Reach out to the committee on lgbt@soton.ac.uk to find out more.`, guild.Name)

		b.State.SendMessageComplex(memberChannel.ID, *message)
	})
}

func (b *Bot) findInvalidMembersInGuild(guildID discord.GuildID, memberTypeForGuild StudentType, runForEachInvalidMember func(discord.User, string)) {
	memberList, err := b.State.Members(guildID)
	if err != nil {
		log.Fatalln("Failed fetching member list from guild", guildID, "with error", err)
	}

	for _, member := range memberList {
		if member.User.Bot {
			// don't warn or remove bots!
			continue
		}

		authenticated, userType := isDiscordAuthenticated(member.User, memberTypeForGuild)
		if !authenticated {
			runForEachInvalidMember(member.User, userType)
		}
	}
}
