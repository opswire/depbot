FROM alpine:3.18 AS builder
RUN echo "build stage"

FROM scratch AS empty

FROM --platform=linux/amd64 nginx:1.21-alpine
COPY --from=builder /out /

# Stage alias should not be re-detected as image:
FROM builder AS final

# Templated image, must be skipped:
FROM ${BASE_IMAGE}

# Non-semver tag, skipped (Occurrence not produced):
FROM ubuntu:latest
