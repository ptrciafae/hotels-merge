package hotels

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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

func GetHotels() (*Hotels, error) {
	// NOTE: I would cache the results for a certain time period to avoid fetching from suppliers on every request
	// especially since the data are not likely to change very often (list of hotels)

	responses := make([][]byte, 0)
	for _, supplier := range GetSuppliers() {
		body, err := fetchSupplierData(supplier.Name, supplier.URL)
		if err != nil {
			fmt.Println("Error fetching supplier data:", err)
			continue
		}
		responses = append(responses, body)
	}

	return deduplicateHotels(responses)
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

func deduplicateHotels(hotelsList [][]byte) (*Hotels, error) {
	// Implementation for deduplicating hotels based on name and location

	return &Hotels{}, nil
}
