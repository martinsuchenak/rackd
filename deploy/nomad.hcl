job "rackd" {
  datacenters = ["dc1"]
  type        = "service"

  group "rackd" {
    count = 1

    network {
      port "http" {
        static = 8080
      }
    }

    volume "data" {
      type      = "host"
      source    = "rackd-data"
      read_only = false
    }

    task "rackd" {
      driver = "docker"

      config {
        image = "ghcr.io/martinsuchenak/rackd:latest"
        ports = ["http"]
      }

      volume_mount {
        volume      = "data"
        destination = "/data"
      }

      env {
        DATA_DIR    = "/data"
        LISTEN_ADDR = ":8080"
        LOG_FORMAT  = "json"
        LOG_LEVEL   = "info"
      }

      template {
        data = <<-EOF
          {{ with nomadVar "nomad/jobs/rackd" }}
          API_AUTH_TOKEN={{ .api_auth_token }}
          MCP_AUTH_TOKEN={{ .mcp_auth_token }}
          {{ end }}
        EOF
        destination = "secrets/env"
        env         = true
      }

      resources {
        cpu    = 256
        memory = 256
      }

      service {
        name = "rackd"
        port = "http"

        check {
          type     = "http"
          path     = "/api/datacenters"
          interval = "30s"
          timeout  = "5s"
        }

        tags = [
          "traefik.enable=true",
          "traefik.http.routers.rackd.rule=Host(`rackd.example.com`)",
        ]
      }
    }
  }
}
