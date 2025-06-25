package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"opentable-monitor/monitor"
	"opentable-monitor/notifications"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
)

func themed(f *huh.Form) *huh.Form { return f.WithTheme(huh.ThemeCharm()) }

//  Label helpers

func trunc(s string, n int) string {
	if len(s) > n {
		return s[:n-1] + "…"
	}
	return s
}

func menuLabel(r monitor.AutoResult) string {
	return fmt.Sprintf("%-40s | %-20s | %-16s | %-12s | %-10s | %s",
		trunc(r.Name, 40),
		trunc(r.Neighborhood, 20),
		trunc(r.Metro, 16),
		r.Country,
		r.Type,
		r.ID,
	)
}

func confirmLabel(r monitor.AutoResult) string {
	return strings.Join([]string{
		r.Name, r.Neighborhood, r.Metro, r.Country, r.Type, r.ID,
	}, " | ")
}

//  Time-slot builder

func buildTimes() []string {
	times := make([]string, 0, 48)
	for h := range 24 {
		for _, m := range []int{0, 30} {
			times = append(times, fmt.Sprintf("%02d:%02d", h, m))
		}
	}
	return times
}

//  Main loop

func main() {
	_ = godotenv.Load()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Get Discord webhook URL from environment variable
	webhookURL := os.Getenv("DISCORD_WEBHOOK_URL")
	if webhookURL == "" {
		log.Fatalf("DISCORD_WEBHOOK_URL environment variable is required")
	}

	// Initialize Discord notifier
	discord := notifications.NewDiscordNotifier(webhookURL)

	cli, err := monitor.New(ctx)
	if err != nil {
		log.Fatalf("monitor init: %v", err)
	}

	for {
		// search term
		var term string
		if err := themed(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("🔍  Restaurant search (blank = quit)").
					Placeholder("House of Prime Rib").
					Value(&term),
			))).Run(); err != nil || strings.TrimSpace(term) == "" {
			fmt.Println("Bye!")
			return
		}

		// fetch results
		var results []monitor.AutoResult
		var fetchErr error

		_ = spinner.New().
			Title("Fetching results…").
			Context(ctx).
			Action(func() {
				results, fetchErr = cli.Autocomplete(ctx, term)
			}).
			Run()

		if fetchErr != nil {
			log.Printf("autocomplete: %v\n", fetchErr)
			continue
		}
		if len(results) == 0 {
			fmt.Println("No matches – try again.")
			continue
		}

		// restaurant select
		resOpts := make([]huh.Option[string], 0, len(results)+1)
		for _, r := range results {
			resOpts = append(resOpts, huh.NewOption(menuLabel(r), r.ID))
		}
		resOpts = append(resOpts, huh.NewOption("🔄  New search", "redo"))

		var pickedID string
		if err := themed(huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("Select restaurant (↑/↓, ⏎)").
					Options(resOpts...).
					Height(12).
					Value(&pickedID),
			))).Run(); err != nil {
			fmt.Println("Selection aborted.")
			return
		}
		if pickedID == "redo" {
			continue
		}

		var picked monitor.AutoResult
		for _, r := range results {
			if r.ID == pickedID {
				picked = r
				break
			}
		}

		// date select
		var datePref string
		if err := themed(huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("🗓️  Reservation date (YYYY-MM-DD)").
					Placeholder(time.Now().Format("2006-01-02")).
					Validate(func(v string) error {
						v = strings.TrimSpace(v)
						if _, err := time.Parse("2006-01-02", v); err != nil {
							return fmt.Errorf("invalid date format")
						}
						return nil
					}).
					Value(&datePref),
			))).Run(); err != nil {
			fmt.Println("Selection aborted.")
			return
		}

		// time-slot select
		timeOpts := buildTimes()
		tsOpts := make([]huh.Option[string], len(timeOpts))
		for i, t := range timeOpts {
			tsOpts[i] = huh.NewOption(t, t)
		}

		var timePref string
		if err := themed(huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().
					Title("⏰  Preferred reservation time (24-hour)").
					Description("We'll ping you for the closest available slots.").
					Options(tsOpts...).
					Height(10).
					Value(&timePref),
			))).Run(); err != nil {
			fmt.Println("Selection aborted.")
			return
		}

		// party size
		var partySizeStr string
		var partySize int

		// Use string for the form
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("👥  Party size").
					Placeholder("2").
					Validate(func(v string) error {
						if _, err := strconv.Atoi(v); err != nil {
							return fmt.Errorf("must be a valid number")
						}
						return nil
					}).
					Value(&partySizeStr), // Use string pointer here
			),
		)

		// After form submission, convert to int
		if err := form.Run(); err == nil {
			partySize, _ = strconv.Atoi(partySizeStr)
		}

		// start-monitor confirmation
		var start bool
		if err := themed(huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Start monitor?").
					Affirmative("Yes").
					Negative("No").
					Value(&start),
			))).Run(); err != nil {
			fmt.Println("Aborted.")
			return
		}
		if !start {
			fmt.Println("\nMonitor cancelled.")
			return
		}

		// final message
		fmt.Printf("\n✅  Monitoring: %s\n", confirmLabel(picked))
		fmt.Printf("   Preferred date : %s\n", datePref)
		fmt.Printf("   Preferred time : %s\n", timePref)
		fmt.Printf("   Party size     : %d\n", partySize)
		fmt.Printf("   Discord webhook configured: %s\n\n", "✅")

		// Send initial monitoring started webhook
		if err := discord.SendMonitoringStarted(picked, datePref, timePref, partySize); err != nil {
			log.Printf("Failed to send start webhook: %v", err)
		}

		// START THE BACKGROUND MONITOR
		// Use a fresh, cancellable context so the monitor isn't limited to 30 s.
		monitorCtx, stop := context.WithCancel(context.Background())
		defer stop() // ensures cleanup if main returns for any reason

		// Run the monitor inside a spinner for a nicer UX.
		_ = spinner.New().
			Title("Running reservation monitor… (Ctrl-C to quit)").
			Context(monitorCtx).
			Action(func() {
				// Use the modified StartMonitor with callback
				if err := cli.StartMonitor(
					monitorCtx,
					picked.ID,
					datePref,
					timePref,
					partySize,
					func(exactMatch bool, reservationURL string, alternativeTimes []string) {
						if exactMatch {
							// Send exact slot found webhook
							if err := discord.SendSlotFound(picked, datePref, timePref, partySize, reservationURL); err != nil {
								log.Printf("Failed to send slot found webhook: %v", err)
							}
						} else if len(alternativeTimes) > 0 {
							// Send alternative times webhook
							if err := discord.SendAlternativeTimes(picked, datePref, partySize, alternativeTimes, reservationURL); err != nil {
								log.Printf("Failed to send alternative times webhook: %v", err)
							}
						}
					},
				); err != nil {
					// Send error notification
					if err := discord.SendError(picked, err.Error()); err != nil {
						log.Printf("Failed to send error webhook: %v", err)
					}
					log.Printf("monitor: %v\n", err)
				}
			}).
			Run()

		// Send monitoring stopped notification
		if err := discord.SendMonitoringStopped(picked, "Monitor completed or cancelled"); err != nil {
			log.Printf("Failed to send stop webhook: %v", err)
		}

		// After monitor exits (slot found or ctx cancelled) we're done.
		return
	}
}
