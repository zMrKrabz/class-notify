package class_notify

import (
	"context"
	"errors"
	"fmt"
	"github.com/zMrKrabz/class-notify/schools"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
)

type Database struct {
	collection *mongo.Collection
}

func (db *Database) Connect(uri string) error {
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(uri))
	if err != nil {
		return fmt.Errorf("could not connect to data base: %s", err)
	}
	db.collection = client.Database("main").Collection("classes")

	if indexName, err := db.collection.Indexes().CreateOne(context.Background(), mongo.IndexModel{
		Keys:    bson.D{{Key: "uri", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil {
		return fmt.Errorf("creating unique index for uri field with indexName %s: %s",
			indexName, err)
	}
	log.Println("connected to database collection successfully")

	return nil
}

func (db *Database) GetAllEvents(c chan Event) error {
	cursor, err := db.collection.Find(context.TODO(), bson.D{})
	if err != nil {
		return fmt.Errorf("getting collection cursor: %s", err)
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var result Event
		if err := cursor.Decode(&result); err != nil {
			return fmt.Errorf("decdoing result: %s", err)
		}
		c <- result
	}
	return nil
}

func (db *Database) GetAllActiveEvents(c chan Event) error {
	// active events are events that are actively monitored, which are those that are not completed
	filter := bson.D{{
		"status", bson.D{{"$ne", schools.COMPLETED}},
	}}
	cursor, err := db.collection.Find(context.TODO(), filter)
	if err != nil {
		return fmt.Errorf("getting collection cursor: %s", err)
	}
	defer cursor.Close(context.TODO())

	for cursor.Next(context.TODO()) {
		var result Event
		if err := cursor.Decode(&result); err != nil {
			return fmt.Errorf("decdoing result: %s", err)
		}
		c <- result
	}
	return nil
}

func (db *Database) GetActiveEventsCount() (int64, error) {
	filter := bson.D{{
		"status", bson.D{{"$ne", schools.COMPLETED}},
	}}
	count, err := db.collection.CountDocuments(context.TODO(), filter)
	if err != nil {
		return 0, fmt.Errorf("failed to count documents: %s", err)
	}
	return count, nil
}

var ErrNoSuchEvent = errors.New("class_notify: no classes exist with such uri")

func (db *Database) GetEventWithURI(uri string) (Event, error) {
	filter := bson.D{{"uri", uri}}
	result := db.collection.FindOne(context.TODO(), filter)
	if result.Err() != nil {
		if errors.Is(result.Err(), mongo.ErrNoDocuments) {
			return Event{}, ErrNoSuchEvent
		}
		return Event{}, fmt.Errorf("finding event with uri %s: %s", uri, result.Err())
	}

	var e Event
	if err := result.Decode(&e); err != nil {
		return Event{}, fmt.Errorf("decoding result %s", err)
	}
	return e, nil
}

func (db *Database) GetEventsWithSubscriber(userID string) ([]Event, error) {
	filter := bson.D{{"subscribers", userID}}
	cursor, err := db.collection.Find(context.TODO(), filter)
	if err != nil {
		return nil, fmt.Errorf("getting cursor with filter %s: %s", filter, err)
	}

	var events []Event
	if err := cursor.All(context.TODO(), &events); err != nil {
		return nil, fmt.Errorf("decoding results as events: %s", err)
	}
	return events, nil
}

func (db *Database) CreateEvent(uri string, details schools.ClassDetails, userID string) (Event, error) {
	subscribers := make([]string, 1)
	subscribers[0] = userID
	event := Event{
		URI:          uri,
		ClassDetails: details,
		Subscribers:  subscribers,
	}

	result, err := db.collection.InsertOne(context.TODO(), event)
	if err != nil {
		return Event{}, fmt.Errorf("inserting event %s: %s", event, err)
	}
	log.Printf("created event %s with ID: %s\n", event, result.InsertedID)
	return event, nil
}

func (db *Database) AddSubscriber(uri string, subscriberID string) error {
	filter := bson.D{{"uri", uri}}
	update := bson.D{{"uri", bson.D{{"$push", bson.D{{"subscribers", subscriberID}}}}}}
	result, err := db.collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return fmt.Errorf("update data base with filter %s and update query %s: %s",
			filter, update, err)
	}
	if result.MatchedCount == 0 {
		return errors.New("unable to match any events with uri: " + uri)
	}
	if result.ModifiedCount == 0 {
		return errors.New("unable to update any events with uri: " + uri)
	}
	log.Printf("successfuly added %s to %s event", subscriberID, uri)
	return nil
}

func (db *Database) RemoveSubscriber(uri string, subscriberID string) error {
	filter := bson.D{{"uri", uri}}
	update := bson.D{{"$pull", bson.D{{"subscribers", subscriberID}}}}
	result, err := db.collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return fmt.Errorf("update data base with filter %s and update query %s: %s",
			filter, update, err)
	}
	if result.MatchedCount == 0 {
		return errors.New("failed to match any events with uri: " + uri)
	}
	if result.ModifiedCount == 0 {
		return errors.New("failed to update any events with uri: " + uri)
	}
	log.Printf("successfuly removed %s from %s event\n", subscriberID, uri)
	return nil

}

func (db *Database) RemoveEvent(uri string) error {
	filter := bson.D{{"uri", uri}}
	result, err := db.collection.DeleteOne(context.TODO(), filter)
	if err != nil {
		return fmt.Errorf("DeleteOne with filter %s: %s", filter, err)
	}
	if result.DeletedCount == 0 {
		return errors.New("failed to delete event with uri " + uri)
	}
	log.Println("successfully removed event with uri " + uri)
	return nil
}

func (db *Database) UpdateEventDetails(uri string, details schools.ClassDetails) error {
	filter := bson.D{{"uri", uri}}
	update := bson.D{{"$set", bson.D{{"class_details", details}}}}
	result, err := db.collection.UpdateOne(context.TODO(), filter, update)
	if err != nil {
		return fmt.Errorf("failed to update event using filter %s and update query %s: %s",
			filter, update, err)
	}
	if result.MatchedCount == 0 {
		return fmt.Errorf("matched 0 documents with filter %s and update %s",
			filter, update)
	}
	if result.ModifiedCount == 0 {
		return fmt.Errorf("modified 0 documents with filter %s and update %s",
			filter, update)
	}
	log.Printf("successfully updated event %s with class details %s\n", uri, details)
	return nil
}
