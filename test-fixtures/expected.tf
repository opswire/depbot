resource "kubernetes_deployment" "app" {
  spec {
    template {
      spec {
        container {
          name  = "app"
          image = "myorg/app:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        }
        container {
          name  = "sidecar"
          image = "envoyproxy/envoy:9.9.9@sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
        }
      }
    }
  }
}

# Comment - should be skipped:
# image = "fake/skipped:9.9.9"

// Also a comment:
// image = "another/skipped:8.8.8"

resource "non_semver" {
  image = "ubuntu:latest"
}
