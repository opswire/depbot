resource "kubernetes_deployment" "app" {
  spec {
    template {
      spec {
        container {
          name  = "app"
          image = "docker.io/myorg/app:3.5.0"
        }
        container {
          name  = "sidecar"
          image = "gcr.io/envoyproxy/envoy:3.28.0"
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
  image = "docker.io/library/ubuntu:latest"
}

resource "short_form" {
  image = "nginx:1.21"
}
