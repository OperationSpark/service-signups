// Package greenlight contains DTOs from Greenlight's Mongo database.
package greenlight

import (
	"net/url"
	"strings"
	"time"
)

type (
	Geometry struct {
		Lat float64 `json:"lat"`
		Lng float64 `json:"lng"`
	}
	GooglePlace struct {
		PlaceID  string   `json:"placeId"`
		Name     string   `json:"name"`
		Address  string   `json:"address"`
		Phone    string   `json:"phone"`
		Website  string   `json:"website"`
		Geometry Geometry `json:"geometry"`
	}

	Location struct {
		ID          string      `bson:"_id"`
		GooglePlace GooglePlace `bson:"googlePlace"`
	}

	Session struct {
		ID           string    `bson:"_id"`
		CreatedAt    time.Time `bson:"createdAt"`
		Cohort       string    `bson:"cohort"`
		LocationID   string    `bson:"locationId"`
		LocationType string    `bson:"locationType"` // TODO: Use enum?
		ProgramID    string    `bson:"programId"`
		Name         string    `bson:"name"`
		Students     []string  `bson:"students"`
		Times        Times     `bson:"times"` // TODO: Check out "inline" struct tag
	}

	Signup struct {
		// Legacy Meteor did not use Mongo's ObjectID() _id creation.
		ID          string    `bson:"_id"`
		SessionID   string    `bson:"sessionId"`
		NameFirst   string    `bson:"nameFirst"`
		NameLast    string    `bson:"nameLast"`
		FullName    string    `bson:"fullName"`
		Cell        string    `bson:"cell"`
		Email       string    `bson:"email"`
		CreatedAt   time.Time `bson:"createdAt"`
		ZoomJoinURL string    `bson:"zoomJoinUrl"`
	}

	Times struct {
		Start struct {
			DateTime time.Time `bson:"dateTime"`
		} `bson:"start"`
	}
)

// ParseAddress returns two strings, location line1 and cityStateZip
// It takes a full address and splits the string into the street address string and a cityStateZip string
func ParseAddress(address string) (line1, cityStateZip string) {
	location := strings.SplitN(address, ",", 2)
	if address == "" {
		return "", ""
	}
	if len(location) == 1 {
		return strings.TrimSpace(location[0]), ""
	}

	return strings.TrimSpace(location[0]), strings.TrimSpace(strings.TrimSuffix(location[1], ", USA"))
}

// GoogleLocationLink returns a google maps link of the input address
// It uses the parseAddress function to split the address up and then url encode the strings to make the url
func GoogleLocationLink(address string) string {
	if address == "" {
		return ""
	}
	line1, cityStateZip := ParseAddress(address)
	if line1 == "" || cityStateZip == "" {
		return ""
	}
	addressPath := url.QueryEscape(line1 + "," + cityStateZip)
	return "https://www.google.com/maps/place/" + addressPath
}