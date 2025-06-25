package monitor

import (
	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
)

// reusable bits (cookie jar, UA, headers)
var (
	cookieJar      = tls_client.NewCookieJar()
	defaultUA      = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36"
	defaultSecChUA = "\"Google Chrome\";v=\"133\", \"Chromium\";v=\"133\", \"Not/A)Brand\";v=\"24\""
)

// baseHeaders is copied onto each outbound request so we never drift.
func baseHeaders() http.Header {
	return http.Header{
		"accept":                    {"text/html"},
		"accept-language":           {"en-US,en;q=0.9"},
		"cache-control":             {"max-age=0"},
		"priority":                  {"u=0, i"},
		"sec-ch-ua":                 {defaultSecChUA},
		"sec-ch-ua-mobile":          {"?0"},
		"sec-ch-ua-platform":        {"\"macOS\""},
		"sec-fetch-dest":            {"document"},
		"sec-fetch-mode":            {"navigate"},
		"sec-fetch-site":            {"same-origin"},
		"sec-fetch-user":            {"?1"},
		"upgrade-insecure-requests": {"1"},
		"user-agent":                {defaultUA},
		http.HeaderOrderKey: {
			"accept",
			"accept-language",
			"cache-control",
			"priority",
			"sec-ch-ua",
			"sec-ch-ua-mobile",
			"sec-ch-ua-platform",
			"sec-fetch-dest",
			"sec-fetch-mode",
			"sec-fetch-site",
			"sec-fetch-user",
			"upgrade-insecure-requests",
			"user-agent",
		},
	}
}
