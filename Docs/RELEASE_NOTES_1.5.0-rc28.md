# GP-SDR 1.5.0-rc28

- Allows the analog FPV workspace to use any locally connected receiver whose reported tuning range covers the selected frequency.
- Makes FPV and the main receiver workspaces capability-aware: HackRF-only LNA, VGA, RF amplifier, and antenna-power controls are hidden for other radios.
- Adds a PlutoSDR connection selector to Hardware for choosing an available USB or Ethernet/network path.
- Repairs narrow and wide Settings layouts so cards, forms, long paths, release notes, and controls no longer overflow or collapse into broken boxes.
- Documents analog FPV reception and PlutoSDR transport selection in the project wiki.
- Shows raw digital decoder evidence in each expanded Mapper result instead of reducing it to a generic detected state.
- Adds manual Mapper result confirmation, feeding verified modulation/protocol labels into the existing local learning library.
