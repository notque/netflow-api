package data

// Event contains high-level data about an event, intended as a list item
type Event struct {
	ID   string `json:"event_id"`
	Type string `json:"event_type"`
	Time string `json:"event_time"`
}

// Event list for returning in the API
type EventList struct {
	NextURL string  `json:"next"`
	PrevURL string  `json:"previous"`
	Events  []Event `json:"events"`
	Total   int     `json:"total"`
}

// EventDetail contains the CADF payload, enhanced with names for IDs
type EventDetail struct {
	PublisherID string `json:"publisher_id"`
	EventType   string `json:"event_type"`
	Payload     struct {
		Observer struct {
			TypeURI string `json:"typeURI"`
			ID      string `json:"id"`
		} `json:"observer"`
		ResourceInfo string `json:"resource_info"`
		TypeURI      string `json:"typeURI"`
		Initiator    struct {
			TypeURI     string `json:"typeURI"`
			DomainID    string `json:"domain_id,omitempty"`
			DomainName  string `json:"domain_name,omitempty"`
			ProjectID   string `json:"project_id,omitempty"`
			ProjectName string `json:"project_name,omitempty"`
			UserID      string `json:"user_id"`
			UserName    string `json:"user_name"`
			Host        struct {
				Agent   string `json:"agent"`
				Address string `json:"address"`
			} `json:"host"`
			ID string `json:"id"`
		} `json:"initiator"`
		EventTime string `json:"eventTime"`
		Action    string `json:"action"`
		EventType string `json:"eventType"`
		ID        string `json:"id"`
		Outcome   string `json:"outcome"`
		Target    struct {
			TypeURI string `json:"typeURI"`
			ID      string `json:"id"`
			Name    string `json:"name,omitempty"`
		} `json:"target"`
	} `json:"payload"`
	MessageID string `json:"message_id"`
	Priority  string `json:"priority"`
	Timestamp string `json:"timestamp"`
}

type Filter struct {
	source       string
	resourceType string
	resourceName string
	userName     string
	eventType    string
	time         string
	offset       uint32
	limit        uint8
	sort         string
}