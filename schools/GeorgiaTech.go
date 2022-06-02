package schools

import (
	"fmt"
	"github.com/PuerkitoBio/goquery"
	"io"
	"net/http"
	"strconv"
)

type GeorgiaTech struct {}

func (gt *GeorgiaTech) GetClassDetails(uri string) (ClassDetails, error) {
	if err := gt.validate(uri); err != nil {
		return ClassDetails{}, fmt.Errorf("uri is invalid: %s", err)
	}
	resp, err := http.Get(uri)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("getting uri: %s", err)
	}
	defer resp.Body.Close()
	details, err := gt.parse(resp.Body)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("parsing response body: %s", err)
	}
	return details, nil
}

func (gt *GeorgiaTech) validate(uri string) error {
	return nil
}

func (gt *GeorgiaTech) parse(body io.Reader) (ClassDetails, error) {
	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("parsing body into htmly: %s", err)
	}

	className := doc.Find("body > div.pagebodydiv > table:nth-child(2) > tbody > tr:nth-child(1) > th").Text()
	seatsCapText := doc.Find("body > div.pagebodydiv > table:nth-child(2) > tbody > tr:nth-child(2) > td > table > tbody > tr:nth-child(2) > td:nth-child(2)").Text()
	seatsCap, err := strconv.Atoi(seatsCapText)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("could not convert seats capacity text %s to int: %s", seatsCapText, err)
	}
	seatsActualText := doc.Find("body > div.pagebodydiv > table:nth-child(2) > tbody > tr:nth-child(2) > td > table > tbody > tr:nth-child(2) > td:nth-child(3)").Text()
	seatsActual, err := strconv.Atoi(seatsActualText)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("could not convert seats actual text %s to int: %s", seatsActualText, err)
	}
	waitlistCapText := doc.Find("body > div.pagebodydiv > table:nth-child(2) > tbody > tr:nth-child(2) > td > table > tbody > tr:nth-child(3) > td:nth-child(2)").Text()
	waitlistCap, err := strconv.Atoi(waitlistCapText)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("could not convert waitlist capacity text %s to int: %s", waitlistCapText, err)
	}
	waitlistActualText := doc.Find("body > div.pagebodydiv > table:nth-child(2) > tbody > tr:nth-child(2) > td > table > tbody > tr:nth-child(3) > td:nth-child(3)").Text()
	waitlistActual, err := strconv.Atoi(waitlistActualText)
	if err != nil {
		return ClassDetails{}, fmt.Errorf("could not convert waitlist actual text %s to int: %s", waitlistActualText, err)
	}

	var status ClassStatus
	if seatsCap >= seatsActual {
		if waitlistCap > waitlistActual {
			status = WAITLISTED
		} else {
			status = FULL
		}
	} else {
		status = OPENED
	}

	return ClassDetails{
		Name:              className,
		Description:       "",
		Status:            status,
		SeatsTotal:        seatsCap,
		SeatsRemaining:    seatsCap - seatsActual,
		WaitlistTotal:     waitlistCap,
		WaitlistRemaining: waitlistActual,
	}, nil
}