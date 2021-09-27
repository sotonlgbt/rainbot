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

func (b *Bot) warnInvalidUsers(guildID discord.GuildID) {
	guild, err := b.State.Guild(guildID)
	if err != nil {
		log.Fatalln("Failed fetching guild for", guildID, "with error", err)
	}

	memberList, err := b.State.Members(guildID)
	if err != nil {
		log.Fatalln("Failed fetching member list from guild", guildID, "with error", err)
	}

	memberTypeForGuild := getMemberTypeForGuild(guildID)

	for _, member := range memberList {
		authenticated, userType := isDiscordAuthenticated(member.User, memberTypeForGuild)
		if !authenticated {
			log.Println("User", member.User.Username, "is not correctly authenticated - messaging")
			if !warnInvalidDryRunMode {
				var messageToSend bytes.Buffer
				warningText.Execute(&messageToSend, UserWarningInformation{guild.Name, warnInvalidDeadline, GetStudentTypeFromCode(userType), memberTypeForGuild})

				memberChannel, err := b.State.CreatePrivateChannel(member.User.ID)
				if err != nil {
					log.Println("Failed creating message channel to user", member.User.Username, "with error", err)
					continue
				}

				b.State.SendMessageComplex(memberChannel.ID, api.SendMessageData{
					Content: messageToSend.String(),
					Components: []discord.Component{
						&discord.ActionRowComponent{
							Components: []discord.Component{
								&discord.ButtonComponent{
									CustomID: "verifyme_button_guild_" + guildID.String(),
									Label:    "Let's get verified!",
									Emoji: &discord.ButtonEmoji{
										Name: "🎉",
									},
									Style: discord.PrimaryButton,
								},
							},
						},
					},
				})
			}
		}
	}

}

func (b *Bot) PurgeInvalidUsers(guildID discord.GuildID) {
	guild, err := b.State.Guild(guildID)
	if err != nil {
		log.Fatalln("Failed fetching guild for", guildID, "with error", err)
	}

	memberList, err := b.State.Members(guildID)
	if err != nil {
		log.Fatalln("Failed fetching member list from guild", guildID, "with error", err)
	}

	memberTypeForGuild := getMemberTypeForGuild(guildID)

	for _, member := range memberList {
		authenticated, userType := isDiscordAuthenticated(member.User, memberTypeForGuild)
		if !authenticated {
			log.Println("User", member.User.Username, "is not correctly authenticated - purging")
			b.State.Kick(guildID, member.User.ID, api.AuditLogReason("Incorrectly authenticated for this server and an invalid member purge is running - was: "+userType))

			memberChannel, err := b.State.CreatePrivateChannel(member.User.ID)
			if err != nil {
				log.Println("Failed creating message channel to user", member.User.Username, "with error", err)
				continue
			}

			message, err := b.createReinviteMessage(guildID, member.User)
			if err != nil {
				log.Fatalln("Failed creating reinvite message for user", member.User.Username, "with error", err)
			}

			message.Content = fmt.Sprintf(`You weren't verified for the %s server for this academic year, so we've had to say goodbye for now. Need to reverify? Hit the button below.
					No longer the right server for you? There's other opportunities! Reach out to the committee on lgbt@soton.ac.uk to find out more.`, guild.Name)

			b.State.SendMessageComplex(memberChannel.ID, *message)
		}
	}
}
