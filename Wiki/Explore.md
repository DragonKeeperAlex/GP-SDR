# Explore recorded data

Open **Review → Explore** to inspect a snapshot of the loaded activity journal without opening an SDR. Press **Refresh** to include newly collected results.

## Filter and inspect

Filter by text, UTC date range, frequency range in MHz, receive area, or modulation. Press Refresh to apply. Frequency groups combine the same frequency within approximately 0.1-degree receive areas; recordings without coordinates remain separate from located recordings.

- **Hourly heatmap:** recorded event counts by frequency/area and UTC hour, for the first 60 sorted groups. Hover for counts; click a row to inspect captures.
- **Daily timeline:** recorded events over the latest 500 observed days. Empty intervals do not prove that a channel was monitored and quiet.
- **Collection map:** an offline coordinate plot. Circles show receiver observation positions; purple diamonds show applicable imported reference-area centers. It is not a street map or transmitter-location estimate.
- **Frequency groups:** sort by frequency, events, recency, or decoded events. Results are paged in groups of 100. Click a frequency to see evidence, nearby reference candidates, transcripts, and up to 50 latest captures across all dates.

The overview statistics and daily timeline summarize the main filters. The Bookmarked checkbox further restricts the group table, heatmap, and map only.

## Organize

Star groups to bookmark them, add a short tag in the detail panel, or save the current main filters as a view. These preferences are stored in this browser origin, not in exported radio profiles or synchronized between devices. A change of server address/port can expose a different preference store. Up to 30 views are supported.

## Interpret the evidence carefully

Explore covers the loaded journal window (up to 25,000 events), excludes simulated events, and does not scan old files independently. Recorded duration is the sum of capture durations and can include overlap; it is not monitoring time. Audio/IQ counts count journal links, not verified existing files. Missing or removed recordings can therefore appear in history without being playable.

Nearby reference candidates require an exact frequency match plus a recorded position inside the imported reference area's radius. A frequency/area match alone does not confirm who transmitted. Records lacking coordinates cannot receive geographic matches. No new RadioReference download or account is required to explore existing data.

True occupancy, timestamped quiet checks, street basemaps, map time playback, and receiver/job-specific Explore filters remain future work. Collection locations are sensitive; normal server access controls apply, and this page does not upload them to an external map provider.

## Transcription hardening

New transcription attempts skip essentially digital silence and conservative stationary broadband-static cases. Standalone non-speech captions such as bracketed water or engine noises are removed. Quiet speech is not rejected merely for low volume. This is a heuristic safeguard, not a guarantee against model hallucinations; old transcripts are not rewritten. Inspect original audio and decoder evidence before treating a transcript as confirmed.
