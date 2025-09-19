An RSS Reader and bookmark manager written in Go and HTMX.

podman build -t rss-reader .

podman run -p 8080:8080 -v .:/app/data rss-reader