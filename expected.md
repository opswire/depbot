# Deployment Notes

This service is built on top of `docker.io/library/alpine:3.99.0@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` and pulls
dependencies from the `gcr.io/myorg/app:3.99.0@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` image at build time.

The internal proxy uses `quay.io/jetstack/cert-manager:3.99.0@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa` for routing.

## Excluded references

# This entire commented line should be skipped: docker.io/pretend/image:9.9.9

Non-semver tags should not be picked up: docker.io/library/ubuntu:latest is
mentioned but won't be matched.

Short-form refs are also rejected: nginx:1.21 doesn't have an explicit domain.

## Host:port should be ignored

Connect to `registry.example.com:5000` for private images.
