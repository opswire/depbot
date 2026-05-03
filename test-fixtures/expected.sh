#!/bin/bash
# Build script
set -e

REGISTRY=registry.example.com:5000
docker pull alpine:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
docker run --rm myorg/app:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa ./entrypoint.sh

# Comment, skipped:
# docker pull skipped/image:9.9.9

# Non-semver, skipped:
docker pull ubuntu:latest

echo "Done with nginx:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa deployment"
