# GP-SDR release checklist

Use this checklist for every public release. A release is not complete when the
packages work but the installation and Wiki instructions still describe the
previous interface.

## Version and source

- Update the server version and release notes.
- Update versioned example filenames in `README.md`, `Docs/INSTALL.md`, the
  packaged Windows README, and `Wiki/Getting-Started.md`.
- Confirm `LICENSE`, `NOTICE`, `THIRD_PARTY.md`, and bundled source archives
  match the components in the packages.

## Documentation gate

- Add every new page, control, setting, decoder, and changed default to the
  appropriate file under `Wiki/`.
- Update `Wiki/_Sidebar.md` when adding or renaming a page.
- Update setup instructions when a dependency, executable name, driver path,
  command-line flag, service unit, firewall rule, or storage location changes.
- Keep newcomer instructions task-oriented: install, first signal, first
  decoder, server setup, verification, and recovery.
- State whether a component is bundled, automatically installable, or manual on
  each supported platform. Never describe process startup or RF energy as a
  successful decode.
- Publish the complete `Wiki/` directory to the live `GP-SDR.wiki` repository
  and verify every sidebar link after publication.

## Verification

- Run the Go tests, race tests, static checks, and web JavaScript syntax check.
- Build every release target and verify `SHA256SUMS.txt`.
- Verify the universal macOS architectures and deep signature.
- Open the packaged app or server, check its reported version, and confirm the
  Hardware, Setup, and Decoder pages match the documentation.
- Record hardware and over-the-air tests separately from build and UI tests.

## Publication

- Publish source, release notes, and all expected assets.
- Wait for GitHub Actions to finish successfully.
- Verify the release tag targets the intended commit and every download is
  present.
- Verify the live Wiki Home page, sidebar, Getting Started, Server Setup, and
  Optional Components pages.
