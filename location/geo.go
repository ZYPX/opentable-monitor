// geo/geo.go
package geo

import (
	"encoding/json"
	"fmt"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
)

// Coordinates is the outbound JSON shape: {"lat": 12.34, "lon": 56.78}
type Coordinates struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// GetCoordinates fetches ipapi.co and returns the JSON document with
// only the latitude & longitude.
func GetCoordinates() (Coordinates, error) {

	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(15),
		tls_client.WithClientProfile(profiles.Chrome_133),
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
	if err != nil {
		return Coordinates{}, fmt.Errorf("init http client: %w", err)
	}

	req, err := http.NewRequest(http.MethodGet, "https://ipapi.co/json", nil)
	if err != nil {
		return Coordinates{}, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("user-agent",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6_1) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/137.0.0.0 Safari/537.36")

	resp, err := client.Do(req)
	if err != nil {
		return Coordinates{}, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	var payload struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return Coordinates{}, fmt.Errorf("decode json: %w", err)
	}

	return Coordinates{Lat: payload.Latitude, Lon: payload.Longitude}, nil
}
