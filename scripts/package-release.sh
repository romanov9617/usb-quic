#!/bin/sh
set -eu

version="${1:?usage: package-release.sh VERSION}"
release_dir="dist/release"
ldflags="-s -w -X usb-quic/internal/adapter/delivery/cli.version=${version} -X usb-quic/internal/adapter/delivery/daemon.version=${version}"

rm -rf "${release_dir}"
mkdir -p "${release_dir}"

for arch in amd64 arm64; do
	name="usb-quic_${version}_linux_${arch}"
	stage="${release_dir}/${name}"

	mkdir -p "${stage}/config" "${stage}/docs" "${stage}/systemd"

	CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -trimpath -ldflags "${ldflags}" -o "${stage}/usb-quic" ./cmd/usb-quic
	CGO_ENABLED=0 GOOS=linux GOARCH="${arch}" go build -trimpath -ldflags "${ldflags}" -o "${stage}/usb-quicd" ./cmd/usb-quicd

	cp CHANGELOG.md LICENSE README.md README.ru.md "${stage}/"
	cp docs/RELEASING.md "${stage}/docs/"
	cp scripts/install.sh scripts/uninstall.sh "${stage}/"
	cp packaging/config/client.env.example packaging/config/server.env.example "${stage}/config/"
	cp packaging/systemd/usb-quic-client.service packaging/systemd/usb-quic-server.service "${stage}/systemd/"

	tar \
		--sort=name \
		--owner=0 \
		--group=0 \
		--numeric-owner \
		-C "${release_dir}" \
		-czf "${release_dir}/${name}.tar.gz" \
		"${name}"
	rm -rf "${stage}"
done

(
	cd "${release_dir}"
	sha256sum *.tar.gz > SHA256SUMS
)
