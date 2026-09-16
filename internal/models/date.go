package models

import (
	"time"
	_ "time/tzdata" // Keep the NBP calendar available in desktop installations.
)

var nbpLocation = func() *time.Location {
	location, err := time.LoadLocation("Europe/Warsaw")
	if err != nil {
		panic(err)
	} // Embedded tzdata includes this zone.
	return location
}()

func NBPToday() string { return time.Now().In(nbpLocation).Format("2006-01-02") }
