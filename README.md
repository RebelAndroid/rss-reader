An RSS Reader and bookmark manager written in Go and HTMX.

podman build -t rss-reader .

podman run -p 8080:8080 -v .:/app/data rss-reader

# example to generate systemd service
podman run --replace --name rss-reader -p 8080:8080 -v .:/app/data rss-reader

podman generate systemd rss-reader > ~/.config/systemd/user/container-naarum.service
