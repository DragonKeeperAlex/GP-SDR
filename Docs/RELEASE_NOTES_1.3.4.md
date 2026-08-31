# GP-SDR 1.3.4

## RTL-SDR USB stability

- Keeps one local `rtl_tcp` helper and one IQ connection open for each active
  RTL-SDR instead of repeatedly opening and closing USB for short captures.
- Serializes Live, Tuner, Mapper, and calibration access to each physical
  receiver while allowing independent receivers to run concurrently.
- Retunes the retained connection and discards a short settling window before
  analysis, preventing stale samples from the previous center frequency.
- Recovers a failed helper once with bounded backoff and preserves its output
  in the reported error instead of discarding the underlying driver message.
- Releases persistent helpers before an idle hardware refresh and during clean
  application shutdown, preventing orphan receiver processes.

The regression was verified on a physical Realtek RTL2838U with an Elonics
E4000 tuner: consecutive 100.1 MHz and 162.55 MHz captures retained the same
helper process and IQ connection.
