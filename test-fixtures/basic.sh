#!/bin/bash
# Build script
set -e

REGISTRY=registry.example.com:5000
docker pull alpine:3.19
docker run --rm myorg/app:1.5.0 ./entrypoint.sh

# Comment, skipped:
# docker pull skipped/image:9.9.9

# Non-semver, skipped:
docker pull ubuntu:latest

echo "Done with nginx:1.21-alpine deployment"
