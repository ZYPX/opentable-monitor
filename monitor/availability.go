package monitor

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"time"

	http "github.com/bogdanfinn/fhttp"
)

type slotInfo struct {
	Time        string
	SlotHash    string
	PointsType  string
	PointsValue int
	Attributes  []string
	IsMandatory bool
}

func (s slotInfo) buildURL(date string, party int, token string, rid int) string {
	dateTime := url.QueryEscape(fmt.Sprintf("%sT%s:00", date, s.Time))
	return fmt.Sprintf(
		"https://www.opentable.ca/booking/details?availabilityToken=%s&dateTime=%s&partySize=%d&points=%d&pointsType=%s&rid=%d&slotHash=%s&isModify=false&isMandatory=%t&cfe=true",
		token, dateTime, party, s.PointsValue, s.PointsType, rid, s.SlotHash, s.IsMandatory,
	)
}

func (c *Client) fetchSlots(
	ctx context.Context,
	ridStr string,
	date string,
	timePref string,
	party int,
) ([]slotInfo, string, int, error) {

	rid, err := strconv.Atoi(ridStr)
	if err != nil {
		return nil, "", 0, fmt.Errorf("restaurant id %q: %w", ridStr, err)
	}

	// build payload
	payload := map[string]any{
		"operationName": "RestaurantsAvailability",
		"variables": map[string]any{
			"onlyPop":      false,
			"forwardDays":  0,
			"requireTimes": false,
			"requireTypes": []string{"Standard", "Experience"},
			"privilegedAccess": []string{
				"VisaDiningProgram", "VisaEventsProgram", "ChaseDiningProgram",
			},
			"restaurantIds":  []int{rid},
			"date":           date,
			"time":           timePref,
			"partySize":      party,
			"databaseRegion": "NA",
		},
		"extensions": map[string]any{
			"persistedQuery": map[string]any{
				"version":    1,
				"sha256Hash": "c056cbf4dbe6a95dbb5f814916415dcff0b2c93c180a456d0d4a3a3f38d0b2cc",
			},
		},
	}
	body, _ := json.Marshal(payload)

	const u = "https://www.opentable.ca/dapi/fe/gql?optype=query&opname=RestaurantsAvailability"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(body))
	if err != nil {
		return nil, "", 0, fmt.Errorf("build req: %w", err)
	}

	// headers
	h := baseHeaders()
	h.Set("accept", "*/*")
	h.Set("content-type", "application/json")
	h.Set("origin", "https://www.opentable.ca")
	h.Set("ot-page-group", "rest-profile")
	h.Set("ot-page-type", "restprofilepage")
	h.Set("priority", "u=1, i")
	h.Set("sec-fetch-dest", "empty")
	h.Set("sec-fetch-mode", "cors")
	h.Set("sec-fetch-site", "same-origin")
	h.Set("x-csrf-token", c.csrf)
	h.Set("x-query-timeout", "5500")
	h[http.HeaderOrderKey] = append(
		h[http.HeaderOrderKey],
		"content-type", "origin", "ot-page-group", "ot-page-type",
		"x-csrf-token", "x-query-timeout",
	)
	req.Header = h

	// request
	resp, err := c.tls.Do(req)
	if err != nil {
		return nil, "", 0, fmt.Errorf("availability request: %w", err)
	}
	defer resp.Body.Close()

	// decode
	var api struct {
		Data struct {
			Availability []struct {
				RestaurantID                int    `json:"restaurantId"`
				RestaurantAvailabilityToken string `json:"restaurantAvailabilityToken"`
				AvailabilityDays            []struct {
					Slots []struct {
						IsAvailable       bool     `json:"isAvailable"`
						TimeOffsetMinutes int      `json:"timeOffsetMinutes"`
						SlotHash          string   `json:"slotHash"`
						PointsType        string   `json:"pointsType"`
						PointsValue       int      `json:"pointsValue"`
						Attributes        []string `json:"attributes"`
						IsMandatory       bool     `json:"isMandatory"`
					} `json:"slots"`
				} `json:"availabilityDays"`
			} `json:"availability"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&api); err != nil {
		return nil, "", 0, fmt.Errorf("decode: %w", err)
	}
	if len(api.Data.Availability) == 0 {
		return nil, "", 0, nil
	}

	ra := api.Data.Availability[0]
	token := ra.RestaurantAvailabilityToken

	// compute real times from offsets
	base, _ := time.Parse("15:04", timePref)
	var list []slotInfo
	for _, day := range ra.AvailabilityDays {
		for _, s := range day.Slots {
			if !s.IsAvailable {
				continue
			}
			tClock := base.Add(time.Minute * time.Duration(s.TimeOffsetMinutes)).Format("15:04")
			list = append(list, slotInfo{
				Time:        tClock,
				SlotHash:    s.SlotHash,
				PointsType:  s.PointsType,
				PointsValue: s.PointsValue,
				Attributes:  s.Attributes,
				IsMandatory: s.IsMandatory,
			})
		}
	}
	return list, token, ra.RestaurantID, nil
}

func hashOfExact(list []slotInfo, pref string) string {
	for _, s := range list {
		if s.Time == pref {
			return s.SlotHash
		}
	}
	return ""
}
