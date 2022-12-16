// Package greenlight contains DTOs from Greenlight's Mongo database.
package greenlight

import "time"

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
		GooglePlace GooglePlace `bson:"googlePlace"`
	}

	Session struct {
		ID        string    `bson:"_id"`
		ProgramID string    `bson:"programId"`
		Times     Times     `bson:"times"` // TODO: Check out "inline" struct tag
		Cohort    string    `bson:"cohort"`
		Students  []string  `bson:"students"`
		Name      string    `bson:"name"`
		CreatedAt time.Time `bson:"createdAt"`
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
