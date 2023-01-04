package meeting

type (
	// https://marketplace.zoom.us/docs/api-reference/zoom-api/methods/#operation/meetingRegistrantCreate
	RegistrantRequest struct {
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		Email       string `json:"email"`
		Address     string `json:"address"`
		City        string `json:"city"`
		State       string `json:"state"`
		Zip         string `json:"zip"`
		Country     string `json:"country"`
		Phone       string `json:"phone"`
		AutoApprove bool   `json:"auto_approve"`
	}

	Occurrence struct {
		Duration     int64  `json:"duration"`
		OccurrenceID string `json:"occurrence_id"`
		StartTime    string `json:"start_time"`
		Status       string `json:"status"`
	}

	RegistrationResponse struct {
		Id           int          `json:"id"`
		JoinURL      string       `json:"join_url"`
		RegistrantID string       `json:"registrant_id"`
		StartTime    string       `json:"start_time"`
		Topic        string       `json:"topic"`
		Occurrences  []Occurrence `json:"occurrences"`
	}
)
