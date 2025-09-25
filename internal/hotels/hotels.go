package hotels

type Hotels []Hotel

type Hotel struct {
	Id                string    `json:"id"`
	DestinationId     int       `json:"destination_id"`
	Name              string    `json:"name"`
	Location          Location  `json:"location,omitempty"`
	Description       string    `json:"description,omitempty"`
	Amenities         Amenities `json:"amenities,omitempty"`
	Images            Images    `json:"images,omitempty"`
	BookingConditions []string  `json:"booking_conditions,omitempty"`
}

type Location struct {
	Lat     float64 `json:"lat,omitempty"`
	Lng     float64 `json:"lng,omitempty"`
	Address string  `json:"address,omitempty"`
	City    string  `json:"city,omitempty"`
	Country string  `json:"country,omitempty"`
}

type Amenities struct {
	General []string `json:"general,omitempty"`
	Room    []string `json:"room,omitempty"`
}

type Images struct {
	Rooms     []Link `json:"rooms,omitempty"`
	Site      []Link `json:"site,omitempty"`
	Amenities []Link `json:"amenities,omitempty"`
}

type Link struct {
	Link        string `json:"link,omitempty"`
	Description string `json:"description,omitempty"`
}
