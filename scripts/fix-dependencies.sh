# scripts/fix-dependencies.sh
#!/bin/bash

echo "Fixing DataScrapexter dependencies and build issues..."

# Add missing Go module dependencies
echo "Adding required Go module dependencies..."
go get golang.org/x/time/rate
go get github.com/fsnotify/fsnotify
go get gopkg.in/yaml.v3
go get github.com/PuerkitoBio/goquery

# Update go.mod with additional dependencies commonly needed for web scraping
echo "Adding additional recommended dependencies..."
go get github.com/gorilla/mux
go get github.com/gorilla/websocket
go get github.com/prometheus/client_golang/prometheus
go get github.com/prometheus/client_golang/prometheus/promhttp
go get github.com/stretchr/testify/assert
go get github.com/stretchr/testify/mock

# Tidy up modules
echo "Tidying Go modules..."
go mod tidy

echo "Dependencies updated successfully!"
echo ""
echo "Updated go.mod should now include:"
echo "- golang.org/x/time/rate (for rate limiting)"
echo "- github.com/fsnotify/fsnotify (for file watching)"
echo "- gopkg.in/yaml.v3 (for YAML parsing)"
echo "- github.com/PuerkitoBio/goquery (for HTML parsing)"
echo "- github.com/gorilla/mux (for HTTP routing)"
echo "- github.com/gorilla/websocket (for WebSocket support)"
echo "- github.com/prometheus/client_golang (for metrics)"
echo "- github.com/stretchr/testify (for testing utilities)"
