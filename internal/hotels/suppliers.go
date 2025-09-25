package hotels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ptrciafae/hotels-merge/internal/mapper"
)

type Suppliers struct {
	Name string
	URL  string
	Data json.RawMessage
}

func GetSuppliers() []Suppliers {
	return []Suppliers{
		{Name: "acme", URL: "https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/acme"},
		{Name: "patagonia", URL: "https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/patagonia"},
		{Name: "paperflies", URL: "https://5f2be0b4ffc88500167b85a0.mockapi.io/suppliers/paperflies"},
	}
}

func FetchAndNormalize(engine *mapper.MappingEngine) (Hotels, error) {
	responses := map[string]json.RawMessage{} // key: supplier name, value: raw JSON data
	for _, supplier := range GetSuppliers() {
		body, err := fetchSupplierData(supplier.Name, supplier.URL)
		if err != nil {
			continue
		}
		responses[supplier.Name] = body
	}

	return deduplicateHotels(responses, engine)
}

func fetchSupplierData(name, url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making GET request to %s: %w", name, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body from %s: %w", name, err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch %s: %s", name, resp.Status)
	}

	return body, nil
}

func deduplicateHotels(hotelsList map[string]json.RawMessage, engine *mapper.MappingEngine) (Hotels, error) {
	normalizedData, err := engine.Transform(hotelsList)
	if err != nil {
		return nil, fmt.Errorf("error transforming data: %w", err)
	}

	var hotels Hotels
	if err := json.Unmarshal(normalizedData, &hotels); err != nil {
		return nil, fmt.Errorf("error unmarshaling normalized data: %w", err)
	}

	fmt.Printf("hotels: %+v\n", hotels)

	return hotels, nil
}
