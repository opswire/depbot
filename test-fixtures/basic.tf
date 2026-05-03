resource "kubernetes_deployment" "app" {
  spec {
    template {
      spec {
        container {
          name  = "app"
          image = "myorg/app:1.5.0"
        }
        container {
          name  = "sidecar"
          image = "envoyproxy/envoy:v1.28.0"
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
