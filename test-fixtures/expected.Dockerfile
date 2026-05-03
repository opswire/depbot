FROM alpine:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa AS builder
RUN echo "build stage"

FROM scratch AS empty

FROM --platform=linux/amd64 nginx:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
COPY --from=builder /out /

# Stage alias should not be re-detected as image:
FROM builder AS final

# Templated image, must be skipped:
FROM ${BASE_IMAGE}

# Non-semver tag, skipped (Occurrence not produced):
FROM ubuntu:latest
