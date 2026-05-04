# Tiltfile for local dev
docker_build('docker.io/myorg/frontend:3.1.0', './frontend')
docker_build("docker.io/myorg/backend:3.2.0", "./backend")
custom_build('gcr.io/myproj/api:3.0.0', 'echo ok', deps=['./api'])

# Skipped, comment:
# docker_build('docker.io/commented/out:9.9.9', '.')

# Bazel-style, image without tag — skipped:
container_pull(
    name = "alpine",
    image = "docker.io/library/alpine",
    tag = "3.18",
)

# Short form, skipped (no explicit domain):
docker_build('myorg/short:3.0.0', '.')

print("hello world")
