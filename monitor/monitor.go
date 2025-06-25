package monitor

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// NotificationCallback is called when slots are found
type NotificationCallback func(exactMatch bool, reservationURL string, alternativeTimes []string)

// StartMonitor polls OpenTable every minute. It calls the callback function
// when the slot set changes: a new slot appears or an old one disappears.
func (c *Client) StartMonitor(
	ctx context.Context,
	restaurantID string,
	date string,
	timePref string,
	partySize int,
	callback NotificationCallback,
) error {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	fmt.Printf("🔎  Watching %s on %s (%s, party %d)…\n",
		restaurantID, date, timePref, partySize)

	// prev holds last-seen slots (key = SlotHash)
	prev := map[string]slotInfo{}

	// onePoll() – returns true when preferred slot found
	onePoll := func() (bool, error) {
		current, token, rid, err := c.fetchSlots(ctx, restaurantID, date, timePref, partySize)
		if err != nil {
			return false, err
		}

		// mapify current for quick lookup
		now := make(map[string]slotInfo, len(current))
		for _, s := range current {
			now[s.SlotHash] = s
		}

		// exact preferred slot?
		exactHash := hashOfExact(current, timePref)
		if slot, ok := now[exactHash]; ok {
			fmt.Printf("\n🎉  Exact slot FOUND — %s at %s\n", date, slot.Time)

			// Call callback for exact match if provided
			if callback != nil {
				reservationURL := slot.buildURL(date, partySize, token, rid)
				callback(true, reservationURL, nil)
			} else {
				// Fallback to console output if no callback
				fmt.Printf("%s\n", slot.buildURL(date, partySize, token, rid))
			}
			return true, nil
		}

		added := []slotInfo{}
		for h, s := range now {
			if _, seen := prev[h]; !seen {
				added = append(added, s)
			}
		}

		removed := []slotInfo{}
		for h, s := range prev {
			if _, still := now[h]; !still {
				removed = append(removed, s)
			}
		}

		if len(added) == 0 && len(removed) == 0 {
			// nothing changed -> stay silent
			return false, nil
		}

		// Handle changes
		if len(prev) == 0 {
			fmt.Printf("\n⏰  Preferred %s unavailable — %d alternative time(s):\n",
				timePref, len(current))
		}

		if len(added) > 0 {
			if len(prev) != 0 { // skip label for first big print
				fmt.Printf("\n➕  %d new slot(s):\n", len(added))
			}

			reservationURL := "" // the first URL (for “Book Now”)
			alternativeTimes := make([]string, 0, len(added))

			for _, s := range added {
				attr := strings.Join(s.Attributes, ",")
				url := s.buildURL(date, partySize, token, rid)

				// terminal output
				fmt.Printf("   • %s [%s] → %s\n", s.Time, attr, url)

				// discord output
				alternativeTimes = append(alternativeTimes,
					fmt.Sprintf("• %s [%s] → [Book](%s)", s.Time, attr, url))

				if reservationURL == "" {
					reservationURL = url
				}
			}

			// forward to caller (Discord) if requested
			if callback != nil && len(alternativeTimes) > 0 {
				callback(false, reservationURL, alternativeTimes)
			}
		}

		if len(removed) > 0 {
			fmt.Printf("\n➖  %d slot(s) disappeared:\n", len(removed))
			for _, s := range removed {
				attr := strings.Join(s.Attributes, ",")
				fmt.Printf("   • %s [%s] (slotHash %s)\n",
					s.Time, attr, s.SlotHash)
			}
		}

		// update cache
		prev = now
		return false, nil
	}

	// first poll (always prints the full list once)
	if ok, err := onePoll(); err != nil || ok {
		return err
	}

	// subsequent polls (every 60 s)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if ok, err := onePoll(); err != nil {
				return err
			} else if ok {
				return nil
			}
		}
	}
}

// StartMonitorConsole is a convenience method that uses console output (backward compatibility)
func (c *Client) StartMonitorConsole(
	ctx context.Context,
	restaurantID string,
	date string,
	timePref string,
	partySize int,
) error {
	return c.StartMonitor(ctx, restaurantID, date, timePref, partySize, nil)
}
