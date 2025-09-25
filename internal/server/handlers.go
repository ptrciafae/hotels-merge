package server

import (
	"encoding/json"
	"net/http"
	"strconv"

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

	id := query.Get("id")
	destinationId := query.Get("destination_id")

	if id == "" && destinationId == "" {
		h.handleGetAllHotels(w, r)
		return
	}

	if id != "" && destinationId != "" {
		http.Error(w, "Only one query parameter (id or destination_id) can be provided at a time", http.StatusBadRequest)
		return
	}

	if id != "" {
		result = h.store.FilterById(id)
	} else if destId := query.Get("destination_id"); destId != "" {
		destIdInt, err := strconv.Atoi(destId)
		if err != nil {
			http.Error(w, "Invalid destination_id", http.StatusBadRequest)
			return
		}
		result = h.store.FilterByDestination(destIdInt)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *Handlers) handleGetAllHotels(w http.ResponseWriter, r *http.Request) {
	result := h.store.GetAll()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
