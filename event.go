package class_notify

import (
	"encoding/json"
	"github.com/zMrKrabz/class-notify/schools"
)

type Event struct {
	URI         string               `bson:"uri"`
	Subscribers  []string             `bson:"subscribers"`
	ClassDetails schools.ClassDetails `bson:"class_details"`
}

func (e Event) String() string {
	b, err := json.Marshal(e)
	if err != nil {
		return ""
	}
	return string(b)
}