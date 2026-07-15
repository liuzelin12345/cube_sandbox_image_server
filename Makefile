CONFIG_FILE := etc/cubesandboximageserver-api.yaml
CONFIG_EXAMPLE := etc/cubesandboximageserver-api.example.yaml

.PHONY: api api-validate config fmt tidy vet test check run

api:
	go tool goctl api go -api cube_sandbox_image_server.api -dir .
	go tool goctl api swagger -api cube_sandbox_image_server.api -dir . -filename yapi

api-validate:
	go tool goctl api validate --api cube_sandbox_image_server.api

config:
	@umask 077; if [ -e "$(CONFIG_FILE)" ]; then \
		echo "$(CONFIG_FILE) already exists; leaving it unchanged"; \
	else \
		cp "$(CONFIG_EXAMPLE)" "$(CONFIG_FILE)"; \
		echo "Created $(CONFIG_FILE); fill in the required credentials before starting the service"; \
	fi

fmt:
	go fmt ./...

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

check: api-validate fmt tidy vet test

run: config
	go run . -f $(CONFIG_FILE)
