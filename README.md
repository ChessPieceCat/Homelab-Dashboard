# Homelab Dashboard 

```text
.
├── cmd/
│   └── dashboard/
│       └── main.go
├── internal/
│   ├── docker/
│   │   ├── client.go
│   │   └── types.go
│   └── http/
│       ├── handlers.go
│       └── router.go
├── web/
│   ├── static/
│   │   ├── css/
│   │   │   └── styles.css
│   │   └── js/
│   │       └── app.js
│   └── templates/
│       └── index.html
├── compose.yaml
├── Dockerfile
└── go.mod
```

## Optional environment variables

- `PORT` (default: `8080`)
- `DOCKER_SOCKET` (default: `/var/run/docker.sock`)
