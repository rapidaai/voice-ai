#!/bin/sh
set -eu

SCRIPT_DIR=$(CDPATH='' cd -- "$(dirname -- "$0")" && pwd)
# shellcheck disable=SC1091
. "$SCRIPT_DIR/load-native-deps.sh"

mode=${1:-builder}
case "$mode" in
  builder) packages='python3 python3-pip libc++-dev libc++abi-dev' ;;
  runtime) packages='libopus0 libopusfile0 libc++1' ;;
  *) echo "usage: $0 [builder|runtime]" >&2; exit 2 ;;
esac

cat > /etc/apt/sources.list.d/debian.sources <<EOF
Types: deb
URIs: https://snapshot.debian.org/archive/debian/${DEBIAN_SNAPSHOT}
Suites: bookworm bookworm-updates
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg

Types: deb
URIs: https://snapshot.debian.org/archive/debian-security/${DEBIAN_SNAPSHOT}
Suites: bookworm-security
Components: main
Signed-By: /usr/share/keyrings/debian-archive-keyring.gpg
EOF

printf 'Acquire::Check-Valid-Until "false";\n' > /etc/apt/apt.conf.d/99snapshot
apt-get update
# shellcheck disable=SC2086
apt-get install -y --no-install-recommends $packages
rm -rf /var/lib/apt/lists/*
