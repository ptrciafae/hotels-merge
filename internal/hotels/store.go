package hotels

import (
	"slices"
	"strconv"
	"strings"
)

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

func (s *HotelStore) FilterByIds(ids string) Hotels {
	idsArr := strings.Split(ids, ",")
	var result Hotels

	for _, h := range s.hotels {
		if slices.Contains(idsArr, strings.TrimSpace(h.Id)) {
			result = append(result, h)
		}
	}

	return result
}

func (s *HotelStore) FilterByDestinations(destinationIds string) Hotels {
	destinationIdsArr := strings.Split(destinationIds, ",")
	var result Hotels
	for _, h := range s.hotels {
		if slices.Contains(destinationIdsArr, strings.TrimSpace(strconv.Itoa(h.DestinationId))) {
			result = append(result, h)
		}
	}
	return result
}
