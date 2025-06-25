package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// AutoResult is the subset of the GraphQL payload we actually care about.
type AutoResult struct {
	ID           string  `json:"id"`
	Type         string  `json:"type"`
	Name         string  `json:"name"`
	Neighborhood string  `json:"neighborhoodName"`
	Metro        string  `json:"metroName"`
	Country      string  `json:"country"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

// Autocomplete returns every matching restaurant for the given term.
// A 30-second context deadline is enforced automatically if the caller
// didn't supply one.
func (c *Client) Autocomplete(ctx context.Context, term string) ([]AutoResult, error) {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) <= 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, 30*time.Second)
		defer cancel()
	}

	raw, err := queryAutocomplete(term, c.tls, c.csrf, c.lat, c.lon)
	if err != nil {
		return nil, err
	}
	return parseAutocomplete(raw)
}

func parseAutocomplete(raw []byte) ([]AutoResult, error) {
	var payload struct {
		Data struct {
			Autocomplete struct {
				Results []AutoResult `json:"autocompleteResults"`
			} `json:"autocomplete"`
		} `json:"data"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return nil, err
	}
	return payload.Data.Autocomplete.Results, nil
}

// queryAutocomplete performs the GraphQL POST.
func queryAutocomplete(
	term string,
	client tls_client.HttpClient,
	csrf string,
	lat, lon float64,
) ([]byte, error) {

	payload := map[string]any{
		"operationName": "Autocomplete",
		"variables": map[string]any{
			"term":          term,
			"latitude":      lat,
			"longitude":     lon,
			"useNewVersion": true,
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "fe1d118abd4c227750693027c2414d43014c2493f64f49bcef5a65274ce9c3c3",
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("marshal payload: %w", err)
	}

	const url = "https://www.opentable.ca/dapi/fe/gql?optype=query&opname=Autocomplete"
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build req: %w", err)
	}

	h := baseHeaders()
	h.Set("accept", "*/*")
	h.Set("content-type", "application/json")
	h.Set("origin", "https://www.opentable.ca")
	h.Set("ot-page-group", "search")
	h.Set("ot-page-type", "multi-search")
	h.Set("priority", "u=1, i")
	h.Set("sec-fetch-dest", "empty")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-site", "same-origin")
	h.Set("x-csrf-token", csrf)
	h.Set("x-query-timeout", "1500")
	h[http.HeaderOrderKey] = append(
		h[http.HeaderOrderKey],
		"content-type",
		"origin",
		"ot-page-group",
		"ot-page-type",
		"x-csrf-token",
		"x-query-timeout",
	)

	req.Header = h

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("autocomplete request: %w", err)
	}
	defer resp.Body.Close()

	return io.ReadAll(resp.Body)
}
