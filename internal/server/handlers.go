package server

import (
	"encoding/json"
	"net/http"

	"github.com/ptrciafae/hotels-merge/internal/hotels"
)

type Handlers struct {
	store *hotels.HotelStore
}

func NewHandlers(store *hotels.HotelStore) *Handlers {
	return &Handlers{store: store}
}

func (h *Handlers) handleQueryHotels(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var result hotels.Hotels

	ids := query.Get("ids")
	destinationIds := query.Get("destination_ids")

	if ids == "" && destinationIds == "" {
		h.handleGetAllHotels(w, r)
		return
	}

	if ids != "" && destinationIds != "" {
		http.Error(w, "Only one query parameter (ids or destination_ids) can be provided at a time", http.StatusBadRequest)
		return
	}

	if ids != "" {
		result = h.store.FilterByIds(ids)
	} else if destinationIds := query.Get("destination_ids"); destinationIds != "" {
		result = h.store.FilterByDestinations(destinationIds)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) handleGetAllHotels(w http.ResponseWriter, r *http.Request) {
	result := h.store.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
