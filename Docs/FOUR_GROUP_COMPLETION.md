# Original four groups — completion ledger

Recovered from the original agreed grouping in “Audit GP-SDR features and bugs”.
The four P25 rebuild milestones in P25_REBUILD_PLAN.md are a separate sub-plan.
Older October 15–17 “Group 3/4” P25 interface/recording notes refer to those P25
milestones, not completion of the original application-wide groups below.
This document restores scope; it does not reset or erase prior evidence.

## Group 1 — Field console, access and data safety

Scope: PLAT-03/04, UI-01–07, MAP-09/10 and DIST-03. Includes password/browser
access, 10-inch touch/OS keyboard, receiver controls and applied state, analog
Tuner/Band Monitor, retention/findings preservation and credentials.

Completed increments: password sign-in accepted by user; OS-level keyboard
correction; user reported touchscreen working; separate browser/box rules;
safer controls and storage visibility/persistence fixes. These do not establish
all page/button or analog-audio acceptance.

Remaining: page-by-page interaction/error/reconnect coverage, analog listening
and field usability, deliberate full/offline/removable/NAS storage behavior.
Use disposable fixtures and preserve original data. The existing long backlog
contains the detailed software versus physical evidence for each item.

## Group 2 — Radios, audio, decoders and receive-only field features

Scope: RF-01–11, DEC-01/02, UI-08/09, CAL-01, VID-01, IMG-01/02.
Includes P25 and multi-receiver allocation, DCS, analyzer, calibrated measurements,
analog FPV and NOAA/SSTV receive workflows. It excludes transmit work.

Completed increments: RF-to-decoder transport demonstrated for Pluto/HackRF/RTL;
P25 replacement session/setup/diagnostics/recordings; bounded receiver/source
settings; persistent transcription and PCM spool recovery. Full P25 milestones
remain open. Native packet identity and single-system browser talkgroup controls
now have bounded software tests; immediate decoder-start failure releases its
audio state and reports the startup cause. Diagnostics exposes the configured
path/control budget. Real RF/audio acceptance is still open.

Remaining: authoritative call/audio boundaries and controls, concurrent-call
software proof and measured capacity, recovery; then independent RF/voice tests
with the large antenna on each receiver. Genuine protocol/image/FPV/calibration
acceptance still requires the appropriate received/reference evidence.

## Group 3 — Mapper, analysis and intelligence

Scope: MAP-01–08/11 and AI-01–07. Includes identification accuracy, payloads,
receiver partitioning/failover, deferred analysis and scheduling, occupancy/
locations, speech/model/classifier/dataset evaluation, Explorer and imports.

Completed increments: partition/preview groundwork; identity reconciliation;
durable home evidence queue; historical metadata batch 256/256; eight checked
raw IQ copies processed on Windows with analyses imported; PC activity guard;
Automatic/Local/Deferred/Off selector with home preference and local fallback.

Remaining: RF-backed accuracy/payload/capacity, failover and scheduling boundary
coverage, retention and resumption across complete workflows, representative
false-positive/abstention measurements. Eight-hour soak was in the original
scope; user now requests bounded checks, so a long soak remains explicitly
unrun rather than silently passed. Metadata LLM results are not raw-RF training.

## Group 4 — Transmit, advanced features, platforms and distribution

Scope: TX-01–04, VID-02, PLAT-01/02, DIST-01/02/04/05/06.
Includes controlled TX/image/modem work, digital FPV, Android/iOS, clean install/
upgrade/rollback, secure updates, dependencies, documentation and signing.

Status: not complete. The Pi-first priority override still holds mobile and
public-release expansion behind core remediation. Preserve prototypes and
release work; do not promote them as tested features. Actual TX acceptance needs
an appropriate attenuated/dummy-load setup; signing/entitlements need their real
prerequisites. No transmit or new mobile/public-release deployment is authorized
by a claim that P25 “milestone 4” is complete.

## Review readiness

Finish reversible core fixes and short, meaningful fixture/live-box checks;
record physical/reference/subjective gates separately. Before declaring readiness
for the requested Astra review, reconcile every original group against stable
IDs in MASTER_REMEDIATION_BACKLOG.md and FEATURE_STATUS_CHECKLIST.md, and supply
an explicit outstanding/held list. A review can evaluate unfinished work, but
must not be presented as proof that all four groups or RF acceptance are done.

Do not replace the master backlog with this summary. Keep calibration, passwords,
settings and recordings, exclude damaged HackRF c5cb, enforce HackRF ≥8 MS/s,
and leave the future tiles/themes redesign deferred unless separately resumed.
