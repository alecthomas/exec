PLATFORMS := "aarch64-linux x86_64-linux aarch64-macos x86_64-macos"

_help:
  @just -l

# Build the intermediaries for the most common platforms
build:
  #!/bin/bash
  set -euo pipefail
  cd intermediary
  rm -f *.gz
  for platform in {{PLATFORMS}}; do
    echo "${platform}"
    out="intermediary-${platform}"
    zig cc -target "${platform}" -s -Oz -o "${out}" intermediary.c
    gzip -9 "${out}"
  done
