# Releasing usb-quic

## Version Policy

Use `v0.x.y` while QUIC requires insecure development TLS or real-hardware
compatibility remains unverified. Publishing `v1.0.0` communicates a stable,
secure, supported contract and should not be used only to mark the first
release.

Recommended first release: `v0.1.0`.

## GitHub Release Procedure

1. Create a release branch from `master`.
2. Confirm the version and user-visible changes in the README.
3. Run:

   ```bash
   go test ./...
   go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run --timeout=5m
   ./scripts/package-release.sh v0.1.0
   (cd dist/release && sha256sum -c SHA256SUMS)
   ```

4. Test installation and removal from both release archives on clean Linux
   `amd64` and `arm64` machines.
5. Test the client/server service configuration with real `usbipd`,
   `usbip-host`, `vhci_hcd`, and at least one physical USB device.
6. Merge the release branch into `master`.
7. Create and push an annotated tag:

   ```bash
   git tag -a v0.1.0 -m "usb-quic v0.1.0"
   git push origin v0.1.0
   ```

8. The `Release` GitHub Actions workflow builds archives, creates
   `SHA256SUMS`, and publishes a GitHub Release.
9. Download the published artifacts, verify checksums, and repeat one smoke
   installation from the public release.

GitHub Releases are sufficient for the first preview. Package repositories
such as Homebrew, AUR, Debian, RPM, or a custom apt repository should wait
until installation layout and configuration are stable.

## Requirements Before v1.0.0

All items below should be complete:

- production TLS with explicit trust configuration and certificate rotation;
- no insecure TLS requirement in normal operation;
- documented threat model and exposed network ports;
- end-to-end tests on supported Linux distributions and kernel versions;
- real-device tests covering list, bind, attach, traffic, detach, and unbind;
- documented behavior when either proxy or the network disconnects;
- stable CLI, daemon flags, configuration files, and systemd units;
- upgrade and rollback procedure;
- signed release artifacts or provenance attestations;
- supported architectures and compatibility matrix;
- resolved decision for `usbip list -d`;
- no known critical or high-severity defects.

When these conditions are met, prepare release notes that explicitly define the
supported contract, then tag and publish `v1.0.0` using the same procedure.
