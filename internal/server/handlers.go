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
	// TODO: more validation on query params

	query := r.URL.Query()
	var result hotels.Hotels
	query.Get("id")

	if id := query.Get("id"); id != "" {
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
