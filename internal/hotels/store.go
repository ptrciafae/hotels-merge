package hotels

type HotelStore struct {
	hotels Hotels
}

func NewHotelStore() *HotelStore {
	return &HotelStore{}
}

func (s *HotelStore) Set(hotels Hotels) {
	s.hotels = hotels
}

func (s *HotelStore) GetAll() Hotels {
	return s.hotels
}

func (s *HotelStore) FilterById(id string) Hotels {
	for _, h := range s.hotels {
		if h.Id == id {
			return Hotels{h}
		}
	}

	return Hotels{}
}

func (s *HotelStore) FilterByDestination(destinationId int) Hotels {
	var result Hotels
	for _, h := range s.hotels {
		if h.DestinationId == destinationId {
			result = append(result, h)
		}
	}
	return result
}
