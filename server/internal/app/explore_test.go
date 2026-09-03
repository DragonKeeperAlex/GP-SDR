package app

import (
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestExploreGroupsLocationAndReferenceCandidates(t *testing.T) {
	date := time.Date(2026, 9, 2, 15, 0, 0, 0, time.UTC)
	events := []TransmissionEvent{
		{FrequencyHz: 155e6, StartedAt: date, DurationSeconds: 2, Location: &ObservationLocation{Latitude: 37.7, Longitude: -121.8}},
		{FrequencyHz: 155e6, StartedAt: date, DurationSeconds: 3, Location: &ObservationLocation{Latitude: 37.71, Longitude: -121.81}},
		{FrequencyHz: 155e6, StartedAt: date, Location: &ObservationLocation{Latitude: 34, Longitude: -118}},
		{FrequencyHz: 155e6, StartedAt: date},
		{FrequencyHz: 155e6, StartedAt: date, Simulated: true},
	}
	profiles := []ScanProfile{{ReferenceArea: &ProfileReferenceArea{Provider: "RadioReference", Latitude: 37.7, Longitude: -121.8, RadiusMiles: 10}, Channels: []ChannelDefinition{{FrequencyHz: 155e6, Name: "Local reference"}}}}
	data := buildExploreData(events, profiles, httptest.NewRequest("GET", "/api/explore", nil))
	if data.Events != 4 || len(data.Rows) != 3 || data.Seconds != 5 || data.LocatedEvents != 3 {
		t.Fatalf("unexpected aggregation: %+v", data)
	}
	if data.Rows[0].Count != 2 || data.Rows[0].Hourly[15] != 2 || len(data.Rows[0].References) != 1 {
		t.Fatalf("local group not matched: %+v", data.Rows[0])
	}
	for _, row := range data.Rows[1:] {
		if len(row.References) > 0 {
			t.Fatal("reference leaked to remote or unknown receive location")
		}
	}
	data = buildExploreData(events, profiles, httptest.NewRequest("GET", "/api/explore?from=2026-09-03", nil))
	if data.Events != 0 {
		t.Fatal("date filter ignored")
	}
	data = buildExploreData(events, profiles, httptest.NewRequest("GET", "/api/explore?min=156", nil))
	if data.Events != 0 {
		t.Fatal("frequency filter ignored")
	}
}

func TestExploreSnapshotIncludesMoreThanRecentAPILimit(t *testing.T) {
	store := &EventStore{}
	for i := 0; i < 2300; i++ {
		store.events = append(store.events, TransmissionEvent{ID: fmt.Sprint(i)})
	}
	if len(store.exploreSnapshot()) != 2300 {
		t.Fatal("explorer truncated to normal timeline API limit")
	}
}
