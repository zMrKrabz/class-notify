package class_notify

import (
	"errors"
	"fmt"
	"github.com/bwmarrin/discordgo"
	"log"
)

type Discord struct {
	session            *discordgo.Session
	registeredCommands []*discordgo.ApplicationCommand
	guildID string
	Bot     *Bot
}

func (d *Discord) Connect(token string, guildID string) error {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return errors.New(fmt.Sprintf("connecting to discord: %s", err))
	}
	d.guildID = guildID
	d.session = s

	s.AddHandler(func(s *discordgo.Session, ready *discordgo.Ready) {
		log.Println("Bot is now running, press CTRL + C to close")
	})

	if err := s.Open(); err != nil {
		return fmt.Errorf("unable to open discord session: %s", err)
	}
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "subscribe",
			Description: "Adds you to the alert list of a class",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "url",
					Description: "url of class to add you to",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        "unsubscribe",
			Description: "Removes you from the alert list of a class",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Name:        "url",
					Description: "url of class to remove you from",
					Type:        discordgo.ApplicationCommandOptionString,
					Required:    true,
				},
			},
		},
		{
			Name:        "classes",
			Description: "Lists all classes you are subscribed to",
		},
	}
	commandHandlers := map[string]func(s *discordgo.Session, i *discordgo.InteractionCreate){
		"subscribe":   d.subscribe,
		"unsubscribe": d.unsubscribe,
		"classes":     d.classes,
	}
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		if h, ok := commandHandlers[i.ApplicationCommandData().Name]; ok {
			h(s, i)
		}
	})

	registeredCommands := make([]*discordgo.ApplicationCommand, len(commands))
	for i, v := range commands {
		cmd, err := s.ApplicationCommandCreate(s.State.User.ID, guildID, v)
		if err != nil {
			fmt.Printf("Could not add command %s because of err: %s", v.Name, err)
		}
		fmt.Printf("registered command %s\n", v.Name)
		registeredCommands[i] = cmd
	}
	d.registeredCommands = registeredCommands

	return nil
}

func (d *Discord) Close() {
	for _, v := range d.registeredCommands {
		err := d.session.ApplicationCommandDelete(d.session.State.User.ID, d.guildID, v.ID)
		if err != nil {
			log.Println(fmt.Sprintf("unable to delete command %s, error: %s", v.Name, err))
		}
	}

	d.session.Close()
}

var ErrUserUnavailable = errors.New("class_notify: unable to create DM with user")

func (d *Discord) UpdateSubscriber(event Event) error {
	for _, s := range event.Subscribers {
		channel, err := d.session.UserChannelCreate(s)
		if err != nil {
			// TODO: on this error, delete user from list of subscribers
			return ErrUserUnavailable
		}
		if _, err := d.session.ChannelMessageSendEmbed(channel.ID, &discordgo.MessageEmbed{
			URL:         event.URI,
			Title:       fmt.Sprintf("CLASS STATUS HAS CHANGED TO %s", event.ClassDetails.Status),
			Description: event.ClassDetails.Name,
		}); err != nil {
			return fmt.Errorf("unable to send message to user: %s", err)
		}
	}
	return nil
}

func (d *Discord) subscribe(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	uri := options[0].StringValue()
	var userID string
	// checks if the interaction was created in a guild or in DMs
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	event, err := d.Bot.Subscribe(uri, userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "unable to add you to event",
			},
		})
		log.Printf("unable to add user %s to event %s: %s\n", userID, uri, err)
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			// TODO pretty this up
			Content: fmt.Sprintf("Added you to class %s", event.ClassDetails.Name),
		},
	})
}

func (d *Discord) unsubscribe(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	uri := options[0].StringValue()
	var userID string
	// checks if the interaction was created in a guild or in DMs
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	event, err := d.Bot.Unsubscribe(uri, userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "unable to subscribe from class",
			},
		})
		log.Printf("unable to unsubsribe user %s from event %s because: %s", userID, uri, err)
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("Unsubscribed from class with name %s", event.ClassDetails.Name),
		},
	})
}

func (d *Discord) classes(s *discordgo.Session, i *discordgo.InteractionCreate) {
	var userID string
	// checks if the interaction was created in a guild or in DMs
	if i.User == nil {
		userID = i.Member.User.ID
	} else {
		userID = i.User.ID
	}
	events, err := d.Bot.GetUserEvents(userID)
	if err != nil {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "unable to list your classes",
			},
		})
		log.Printf("unable to list classes of %s: %s", userID, err)
		return
	}
	s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: fmt.Sprintf("subscribed to %v classes, waiting on embed implementation", len(events)),
		},
	})
}
