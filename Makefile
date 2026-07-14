.PHONY: api api-validate fmt tidy vet test check run

api:
	go tool goctl api go -api cube_sandbox_image_server.api -dir .

api-validate:
	go tool goctl api validate --api cube_sandbox_image_server.api

fmt:
	go fmt ./...

tidy:
	go mod tidy

vet:
	go vet ./...

test:
	go test ./...

check: api-validate fmt tidy vet test

run:
	go run . -f etc/cubesandboximageserver-api.yaml
