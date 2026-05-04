FROM docker.io/library/alpine:3.18 AS builder
RUN echo "build stage"

FROM scratch AS empty

FROM --platform=linux/amd64 gcr.io/distroless/static:3.0.0
COPY --from=builder /out /

# Stage alias should not be re-detected as image:
FROM builder AS final

# Templated image, must be skipped:
FROM ${BASE_IMAGE}

# Non-semver tag, skipped:
FROM docker.io/library/ubuntu:latest

# Short-form, skipped (no explicit domain):
FROM redis:7.0

# Line continuation should be handled (different major - won't update):
FROM \
    quay.io/jetstack/cert-manager:1.13.0
