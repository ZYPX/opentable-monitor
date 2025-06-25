package notifications

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"opentable-monitor/monitor"
)

// Discord webhook structures
type DiscordWebhook struct {
	Content string         `json:"content,omitempty"`
	Embeds  []DiscordEmbed `json:"embeds,omitempty"`
}

type DiscordEmbed struct {
	Title       string              `json:"title,omitempty"`
	Description string              `json:"description,omitempty"`
	Color       int                 `json:"color,omitempty"`
	URL         string              `json:"url,omitempty"`
	Fields      []DiscordEmbedField `json:"fields,omitempty"`
	Footer      *DiscordEmbedFooter `json:"footer,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type DiscordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline,omitempty"`
}

type DiscordEmbedFooter struct {
	Text string `json:"text"`
}

// DiscordNotifier handles Discord webhook notifications
type DiscordNotifier struct {
	webhookURL string
}

// NewDiscordNotifier creates a new Discord notifier with the given webhook URL
func NewDiscordNotifier(webhookURL string) *DiscordNotifier {
	return &DiscordNotifier{
		webhookURL: webhookURL,
	}
}

// SendWebhook sends a webhook to Discord
func (d *DiscordNotifier) SendWebhook(webhook DiscordWebhook) error {
	if d.webhookURL == "" {
		return fmt.Errorf("discord webhook URL not configured")
	}

	jsonData, err := json.Marshal(webhook)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook data: %v", err)
	}

	resp, err := http.Post(d.webhookURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("webhook request failed with status: %d", resp.StatusCode)
	}

	return nil
}

// SendSlotFound sends a notification when the exact preferred slot is found
func (d *DiscordNotifier) SendSlotFound(restaurant monitor.AutoResult, date, timeSlot string, partySize int, reservationURL string) error {
	webhook := DiscordWebhook{
		Content: "🎉 **Reservation Available!**",
		Embeds: []DiscordEmbed{
			{
				Title:       "✅ Exact Time Slot Found!",
				Description: fmt.Sprintf("Your preferred reservation slot is now available at **%s**!", restaurant.Name),
				Color:       0x00FF00, // Green color
				URL:         reservationURL,
				Fields: []DiscordEmbedField{
					{
						Name:   "🏪 Restaurant",
						Value:  restaurant.Name,
						Inline: true,
					},
					{
						Name:   "📍 Location",
						Value:  fmt.Sprintf("%s, %s", restaurant.Neighborhood, restaurant.Metro),
						Inline: true,
					},
					{
						Name:   "🌍 Country",
						Value:  restaurant.Country,
						Inline: true,
					},
					{
						Name:   "📅 Date",
						Value:  date,
						Inline: true,
					},
					{
						Name:   "⏰ Time",
						Value:  timeSlot,
						Inline: true,
					},
					{
						Name:   "👥 Party Size",
						Value:  fmt.Sprintf("%d", partySize),
						Inline: true,
					},
					{
						Name:   "🔗 Book Now",
						Value:  fmt.Sprintf("[Click here to reserve](%s)", reservationURL),
						Inline: false,
					},
				},
				Footer: &DiscordEmbedFooter{
					Text: "OpenTable Monitor • Book quickly before it's taken!",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return d.SendWebhook(webhook)
}

// SendAlternativeTimes sends a notification when alternative times are available
func (d *DiscordNotifier) SendAlternativeTimes(restaurant monitor.AutoResult, date string, partySize int, alternativeTimes []string, reservationURL string) error {
	timesText := strings.Join(alternativeTimes, "\n")
	if len(timesText) > 1000 { // Discord field value limit
		timesText = timesText[:997] + "..."
	}

	webhook := DiscordWebhook{
		Content: "⏰ **Alternative Times Available!**",
		Embeds: []DiscordEmbed{
			{
				Title:       "🔄 Alternative Reservation Times",
				Description: fmt.Sprintf("Your exact preferred time isn't available, but there are other options at **%s**!", restaurant.Name),
				Color:       0xFFAA00, // Orange color
				Fields: []DiscordEmbedField{
					{
						Name:   "🏪 Restaurant",
						Value:  restaurant.Name,
						Inline: true,
					},
					{
						Name:   "📍 Location",
						Value:  fmt.Sprintf("%s, %s", restaurant.Neighborhood, restaurant.Metro),
						Inline: true,
					},
					{
						Name:   "🌍 Country",
						Value:  restaurant.Country,
						Inline: true,
					},
					{
						Name:   "📅 Date",
						Value:  date,
						Inline: true,
					},
					{
						Name:   "👥 Party Size",
						Value:  fmt.Sprintf("%d", partySize),
						Inline: true,
					},
					{
						Name:   "⏰ Available Times",
						Value:  timesText,
						Inline: false,
					},
				},
				Footer: &DiscordEmbedFooter{
					Text: "OpenTable Monitor • Consider booking one of these times!",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return d.SendWebhook(webhook)
}

// SendMonitoringStarted sends a notification when monitoring begins
func (d *DiscordNotifier) SendMonitoringStarted(restaurant monitor.AutoResult, date, preferredTime string, partySize int) error {
	webhook := DiscordWebhook{
		Content: "🔍 **Monitoring Started**",
		Embeds: []DiscordEmbed{
			{
				Title:       "🎯 OpenTable Reservation Monitor Active",
				Description: fmt.Sprintf("Now monitoring **%s** for available reservations!", restaurant.Name),
				Color:       0x5865F2, // Discord blurple
				Fields: []DiscordEmbedField{
					{
						Name:   "🏪 Restaurant",
						Value:  restaurant.Name,
						Inline: true,
					},
					{
						Name:   "📍 Location",
						Value:  fmt.Sprintf("%s, %s", restaurant.Neighborhood, restaurant.Metro),
						Inline: true,
					},
					{
						Name:   "🌍 Country",
						Value:  restaurant.Country,
						Inline: true,
					},
					{
						Name:   "📅 Date",
						Value:  date,
						Inline: true,
					},
					{
						Name:   "⏰ Preferred Time",
						Value:  preferredTime,
						Inline: true,
					},
					{
						Name:   "👥 Party Size",
						Value:  fmt.Sprintf("%d", partySize),
						Inline: true,
					},
				},
				Footer: &DiscordEmbedFooter{
					Text: "OpenTable Monitor • You'll be notified when slots become available!",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return d.SendWebhook(webhook)
}

// SendMonitoringStopped sends a notification when monitoring stops
func (d *DiscordNotifier) SendMonitoringStopped(restaurant monitor.AutoResult, reason string) error {
	webhook := DiscordWebhook{
		Content: "⏹️ **Monitoring Stopped**",
		Embeds: []DiscordEmbed{
			{
				Title:       "🛑 OpenTable Monitor Stopped",
				Description: fmt.Sprintf("Monitoring for **%s** has been stopped.", restaurant.Name),
				Color:       0xFF0000, // Red color
				Fields: []DiscordEmbedField{
					{
						Name:   "🏪 Restaurant",
						Value:  restaurant.Name,
						Inline: true,
					},
					{
						Name:   "📍 Location",
						Value:  fmt.Sprintf("%s, %s", restaurant.Neighborhood, restaurant.Metro),
						Inline: true,
					},
					{
						Name:   "❓ Reason",
						Value:  reason,
						Inline: false,
					},
				},
				Footer: &DiscordEmbedFooter{
					Text: "OpenTable Monitor",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return d.SendWebhook(webhook)
}

// SendError sends an error notification
func (d *DiscordNotifier) SendError(restaurant monitor.AutoResult, errorMsg string) error {
	webhook := DiscordWebhook{
		Content: "❌ **Monitor Error**",
		Embeds: []DiscordEmbed{
			{
				Title:       "⚠️ OpenTable Monitor Error",
				Description: fmt.Sprintf("An error occurred while monitoring **%s**.", restaurant.Name),
				Color:       0xFF0000, // Red color
				Fields: []DiscordEmbedField{
					{
						Name:   "🏪 Restaurant",
						Value:  restaurant.Name,
						Inline: true,
					},
					{
						Name:   "📍 Location",
						Value:  fmt.Sprintf("%s, %s", restaurant.Neighborhood, restaurant.Metro),
						Inline: true,
					},
					{
						Name:   "❌ Error",
						Value:  errorMsg,
						Inline: false,
					},
				},
				Footer: &DiscordEmbedFooter{
					Text: "OpenTable Monitor • Please check the application",
				},
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	}

	return d.SendWebhook(webhook)
}
