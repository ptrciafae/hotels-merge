package hotels

type Hotels []Hotel

type Hotel struct {
	Id                string    `json:"id"`
	DestinationId     int       `json:"destination_id"`
	Name              string    `json:"name"`
	Location          Location  `json:"location"`
	Description       string    `json:"description"`
	Amenities         Amenities `json:"amenities"`
	Images            Images    `json:"images"`
	BookingConditions []string  `json:"booking_conditions"`
}

type Location struct {
	Lat     *float64 `json:"lat"`
	Lng     *float64 `json:"lng"`
	Address string   `json:"address"`
	City    string   `json:"city"`
	Country string   `json:"country"`
}

type Amenities struct {
	General []string `json:"general"`
	Room    []string `json:"room"`
}

type Images struct {
	Rooms     []Link `json:"rooms"`
	Site      []Link `json:"site"`
	Amenities []Link `json:"amenities"`
}

type Link struct {
	Link        string `json:"link"`
	Description string `json:"description"`
}
