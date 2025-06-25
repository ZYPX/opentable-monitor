package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/PuerkitoBio/goquery"
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"

	geo "opentable-monitor/location"
)

// Client bundles the TLS client, CSRF token and user coordinates
// so callers don't repeat expensive setup for every request.
type Client struct {
	tls  tls_client.HttpClient
	csrf string
	lat  float64
	lon  float64
}

// New spins up a ready-to-use *Client in three quick steps:
//  1. create the TLS fingerprinted HTTP client
//  2. fetch a CSRF token
//  3. look up the user's latitude / longitude
func New(ctx context.Context) (*Client, error) {
	tls, err := newTLSClient()
	if err != nil {
		return nil, fmt.Errorf("tls-client: %w", err)
	}
	csrf, err := fetchCSRFToken(tls)
	if err != nil {
		return nil, fmt.Errorf("csrf: %w", err)
	}
	coords, err := geo.GetCoordinates()
	if err != nil {
		return nil, fmt.Errorf("geo lookup: %w", err)
	}

	return &Client{
		tls:  tls,
		csrf: csrf,
		lat:  coords.Lat,
		lon:  coords.Lon,
	}, nil
}

// newTLSClient returns an HTTP/2-capable client with a realistic TLS
// fingerprint and shared cookie jar.
func newTLSClient() (tls_client.HttpClient, error) {
	opts := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(30),
		tls_client.WithClientProfile(profiles.Chrome_133),
		tls_client.WithCookieJar(cookieJar),
		tls_client.WithNotFollowRedirects(),
	}
	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), opts...)
}

// fetchCSRFToken requests the OpenTable homepage and extracts the
// windowVariables.__CSRF_TOKEN__ value from the embedded <script>.
func fetchCSRFToken(client tls_client.HttpClient) (string, error) {
	req, err := http.NewRequest(http.MethodGet, "https://www.opentable.ca/", nil)
	if err != nil {
		return "", fmt.Errorf("build req: %w", err)
	}
	req.Header = baseHeaders()

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(resp.Body); err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	return extractCSRF(buf.Bytes())
}

// extractCSRF parses the primary-window-vars <script> tag.
func extractCSRF(htmlBytes []byte) (string, error) {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(htmlBytes))
	if err != nil {
		return "", err
	}

	jsonTxt := doc.Find("script#primary-window-vars").Text()
	if jsonTxt == "" {
		return "", fmt.Errorf("script tag not found")
	}

	var v struct {
		WindowVariables struct {
			Token string `json:"__CSRF_TOKEN__"`
		} `json:"windowVariables"`
	}
	if err := json.Unmarshal([]byte(jsonTxt), &v); err != nil {
		return "", err
	}
	if v.WindowVariables.Token == "" {
		return "", fmt.Errorf("token missing in windowVariables")
	}
	return v.WindowVariables.Token, nil
}
