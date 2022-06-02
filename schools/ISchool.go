package schools

import "encoding/json"

type ISchool interface {
	GetClassDetails(uri string) (ClassDetails, error)
}

type ClassDetails struct {
	Name              string      `bson:"name"`
	Description       string      `bson:"description"`
	Status            ClassStatus `bson:"status"`
	SeatsTotal        int         `bson:"seats_total"`
	SeatsRemaining    int         `bson:"seats_remaining"`
	WaitlistTotal     int         `bson:"waitlisted_total"`
	WaitlistRemaining int         `bson:"waitlisted_remaining"`
}

func (cd ClassDetails) String() string {
	b, err := json.Marshal(cd)
	if err != nil {
		return ""
	}
	return string(b)
}

type ClassStatus string

const (
	FULL     ClassStatus = "FULL"
	WAITLISTED             = "WAITLISTED"
	OPENED                 = "OPENED"
	COMPLETED              = "COMPLETED" // Term is over, class is no longer in session for given url
)
