package class_notify

import (
	"errors"
	"fmt"
	"github.com/zMrKrabz/class-notify/schools"
	"log"
)

type Bot struct {
	School schools.ISchool
	DB     *Database
}

func (bot *Bot) StartMonitor(updateFn func(event Event) error) {
	for {
		if err := bot.Monitor(updateFn); err != nil {
			log.Printf("error on monitoring: %s\n", err)
		}
	}
}

func (bot *Bot) Monitor(updateFn func(event Event) error) error {
	eventCount, err := bot.DB.GetActiveEventsCount()
	if err != nil {
		return fmt.Errorf("could not get active event count: %s", err)
	}
	log.Printf("checking %d events \n", eventCount)
	events:= make(chan Event, eventCount)
	if err := bot.DB.GetAllActiveEvents(events); err != nil {
		return fmt.Errorf("unable to get active events: %s", err)
	}
	for event := range events {
		log.Printf("checking event %s status", event.URI)
		if err := bot.checkEventStatus(event, updateFn); err != nil {
			log.Printf("failed on checking event %s status: %v\n", event, err)
		}
	}
	return nil
}

func (bot *Bot) checkEventStatus(event Event, updateFn func(event Event) error) error {
	details, err := bot.School.GetClassDetails(event.URI)
	if err != nil {
		return fmt.Errorf("unable to get class details: %s", err)
	}
	if err := bot.DB.UpdateEventDetails(event.URI, details); err != nil {
		return fmt.Errorf("unable to update event with details %s: %s", details, err)
	}
	if event.ClassDetails.Status == details.Status {
		return nil
	}

	if err := updateFn(event); err != nil {
		return fmt.Errorf("unable to send update with function on event: %s", err)
	}
	return nil
}

func (bot *Bot) Subscribe(uri string, userID string) (Event, error) {
	event, err := bot.DB.GetEventWithURI(uri)
	if err != nil {
		if errors.Is(err, ErrNoSuchEvent) {
			event, err := bot.createNewEvent(uri, userID)
			if err != nil {
				return Event{},fmt.Errorf("creating new event with uri %s and usrID %s: %s", uri, userID, err)
			}
			return event, nil
		}
		return Event{}, fmt.Errorf("getting event %s from database: %s", uri, err)
	}
	if err := bot.DB.AddSubscriber(uri, userID); err != nil {
		return Event{}, fmt.Errorf("adding a subscriber with uri %s and userID %s: %s", uri, userID, err)
	}
	log.Printf("Added user %s to event %s", userID, uri)
	return event, nil
}

func (bot *Bot) createNewEvent(uri string, userID string) (Event, error) {
	details, err := bot.School.GetClassDetails(uri)
	if err != nil {
		return Event{}, fmt.Errorf("getting class details of %s", err)
	}

	event, err := bot.DB.CreateEvent(uri, details, userID)
	if err != nil {
		return Event{}, fmt.Errorf("creating new event with details %s: %s",
			details, err)
	}
	log.Printf("created new event: %s\n", event)
	return event, nil
}

func (bot *Bot) Unsubscribe(uri string, userID string) (Event, error) {
	event, err := bot.DB.GetEventWithURI(uri)
	if err != nil {
		return Event{}, fmt.Errorf("unable to get event with uri %s: %s", uri, err)
	}
	if err := bot.DB.RemoveSubscriber(uri, userID); err != nil {
		return Event{}, fmt.Errorf("unable to remove subscriber: %s", err)
	}
	log.Printf("removed user %s from event %s\n", userID, uri)
	return event, nil
}

func (bot *Bot) GetUserEvents(userID string) ([]Event, error) {
	events, err := bot.DB.GetEventsWithSubscriber(userID)
	if err != nil {
		return nil, fmt.Errorf("unalbe to get events: %s", err)
	}
	return events, nil
}
