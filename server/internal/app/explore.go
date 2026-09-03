package app

import (
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

type ExploreReference struct {
	Name          string               `json:"name"`
	Provider      string               `json:"provider"`
	Area          ProfileReferenceArea `json:"area"`
	DistanceMiles float64              `json:"distanceMiles"`
}
type ExploreRow struct {
	Key         string               `json:"key"`
	FrequencyHz float64              `json:"frequencyHz"`
	Location    *ObservationLocation `json:"location,omitempty"`
	Count       int                  `json:"count"`
	Seconds     float64              `json:"seconds"`
	Audio       int                  `json:"audio"`
	IQ          int                  `json:"iq"`
	Decoded     int                  `json:"decoded"`
	Transcribed int                  `json:"transcribed"`
	First       time.Time            `json:"first"`
	Last        time.Time            `json:"last"`
	PeakDBFS    float64              `json:"peakDBFS"`
	Hourly      [24]int              `json:"hourly"`
	Modes       []string             `json:"modes"`
	Callsigns   []string             `json:"callsigns"`
	Label       string               `json:"label"`
	References  []ExploreReference   `json:"references"`
}
type ExploreDay struct {
	Day   string `json:"day"`
	Count int    `json:"count"`
}

func (s *EventStore) exploreSnapshot() []TransmissionEvent {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]TransmissionEvent, len(s.events))
	for i := range items {
		items[i] = s.events[len(s.events)-1-i]
	}
	return items
}

type ExploreData struct {
	Rows          []ExploreRow      `json:"rows"`
	Days          []ExploreDay      `json:"days"`
	Events        int               `json:"events"`
	LoadedEvents  int               `json:"loadedEvents"`
	LocatedEvents int               `json:"locatedEvents"`
	Seconds       float64           `json:"seconds"`
	Locations     map[string]string `json:"locations"`
	Modes         []string          `json:"modes"`
}

func exploreKey(event TransmissionEvent) string {
	if !validExploreLocation(event.Location) {
		event.Location = nil
	}
	return fmt.Sprintf("%.0f|%s", event.FrequencyHz, deferredLocationKey(event))
}
func validExploreLocation(location *ObservationLocation) bool {
	return location != nil && !math.IsNaN(location.Latitude) && !math.IsNaN(location.Longitude) && math.Abs(location.Latitude) <= 90 && math.Abs(location.Longitude) <= 180
}

// The heatmap represents recorded event counts, not scan occupancy: historical
// per-frequency quiet checks and observation durations are not stored here.
func buildExploreData(events []TransmissionEvent, profiles []ScanProfile, request *http.Request) ExploreData {
	q := request.URL.Query()
	search := strings.ToLower(strings.TrimSpace(q.Get("q")))
	start, _ := time.Parse("2006-01-02", q.Get("from"))
	end, _ := time.Parse("2006-01-02", q.Get("to"))
	minMHz, _ := strconv.ParseFloat(q.Get("min"), 64)
	maxMHz, _ := strconv.ParseFloat(q.Get("max"), 64)
	data := ExploreData{Rows: []ExploreRow{}, Days: []ExploreDay{}, Modes: []string{}, Locations: map[string]string{}, LoadedEvents: len(events)}
	groups := map[string]*ExploreRow{}
	days := map[string]int{}
	modes := map[string]bool{}
	references := map[string][]ExploreReference{}
	for _, profile := range profiles {
		if profile.ReferenceArea == nil || !validExploreLocation(&ObservationLocation{Latitude: profile.ReferenceArea.Latitude, Longitude: profile.ReferenceArea.Longitude}) {
			continue
		}
		for _, channel := range profile.Channels {
			key := fmt.Sprintf("%.0f", channel.FrequencyHz)
			references[key] = append(references[key], ExploreReference{Name: channel.Name, Provider: profile.ReferenceArea.Provider, Area: *profile.ReferenceArea})
		}
	}
	for _, event := range events {
		if event.Simulated {
			continue
		}
		if !validExploreLocation(event.Location) {
			event.Location = nil
		}
		locationKey := deferredLocationKey(event)
		data.Locations[locationKey] = deferredLocationLabel(event) + " · " + locationKey
		mode := event.Modulation
		if mode == "" {
			mode = "Unknown"
		}
		modes[mode] = true
		if q.Get("location") != "" && q.Get("location") != locationKey {
			continue
		}
		if q.Get("mode") != "" && q.Get("mode") != mode {
			continue
		}
		if !start.IsZero() && event.StartedAt.Before(start) || !end.IsZero() && !event.StartedAt.Before(end.AddDate(0, 0, 1)) {
			continue
		}
		if minMHz > 0 && event.FrequencyHz < minMHz*1e6 || maxMHz > 0 && event.FrequencyHz > maxMHz*1e6 {
			continue
		}
		haystack := fmt.Sprintf("%.6f %s %s %s %s %s", event.FrequencyHz/1e6, stringValue(event.Label), mode, strings.Join(event.Callsigns, " "), stringValue(event.Transcript), deferredLocationLabel(event))
		if search != "" && !strings.Contains(strings.ToLower(haystack), search) {
			continue
		}
		key := exploreKey(event)
		row := groups[key]
		if row == nil {
			row = &ExploreRow{Key: key, FrequencyHz: event.FrequencyHz, Location: event.Location, First: event.StartedAt, Last: event.StartedAt, PeakDBFS: event.SignalDBFS, References: []ExploreReference{}, Modes: []string{}, Callsigns: []string{}}
			if event.Location != nil {
				for _, ref := range references[fmt.Sprintf("%.0f", event.FrequencyHz)] {
					ref.DistanceMiles = haversineMiles(event.Location.Latitude, event.Location.Longitude, ref.Area.Latitude, ref.Area.Longitude)
					if ref.Area.RadiusMiles > 0 && ref.DistanceMiles <= ref.Area.RadiusMiles && len(row.References) < 10 {
						row.References = append(row.References, ref)
					}
				}
			}
			groups[key] = row
		}
		row.Count++
		row.Seconds += math.Max(0, event.DurationSeconds)
		row.Hourly[event.StartedAt.UTC().Hour()]++
		row.Modes = mergeUniqueStrings(row.Modes, []string{mode})
		row.Callsigns = mergeUniqueStrings(row.Callsigns, event.Callsigns)
		if event.StartedAt.Before(row.First) {
			row.First = event.StartedAt
		}
		if event.StartedAt.After(row.Last) {
			row.Last = event.StartedAt
		}
		if event.SignalDBFS > row.PeakDBFS {
			row.PeakDBFS = event.SignalDBFS
		}
		if row.Label == "" {
			row.Label = stringValue(event.Label)
		}
		if event.AudioPath != nil && *event.AudioPath != "" {
			row.Audio++
		}
		if event.IQPath != nil && *event.IQPath != "" {
			row.IQ++
		}
		if len(event.DecoderMessages) > 0 {
			row.Decoded++
		}
		if strings.TrimSpace(stringValue(event.Transcript)) != "" {
			row.Transcribed++
		}
		data.Events++
		data.Seconds += math.Max(0, event.DurationSeconds)
		if event.Location != nil {
			data.LocatedEvents++
		}
		days[event.StartedAt.UTC().Format("2006-01-02")]++
	}
	for _, row := range groups {
		data.Rows = append(data.Rows, *row)
	}
	sort.Slice(data.Rows, func(i, j int) bool {
		return data.Rows[i].Count > data.Rows[j].Count || data.Rows[i].Count == data.Rows[j].Count && data.Rows[i].Key < data.Rows[j].Key
	})
	for day, count := range days {
		data.Days = append(data.Days, ExploreDay{Day: day, Count: count})
	}
	sort.Slice(data.Days, func(i, j int) bool { return data.Days[i].Day < data.Days[j].Day })
	for mode := range modes {
		data.Modes = append(data.Modes, mode)
	}
	sort.Strings(data.Modes)
	return data
}
