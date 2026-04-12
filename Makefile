.PHONY: server init-super-admin build clean

# Run Commands dynamically (no build required, relies on go run main.go)
server:
	go run main.go server

init-super-admin:
	go run main.go init-super-admin

migrate-up:
	go run main.go migrate up

migrate-down:
	go run main.go migrate down

# Build Single Unified Binary
build:
	mkdir -p bin
	go build -o bin/credchain main.go
	@echo "Unified Binary successfully built to bin/credchain."

clean:
	rm -rf bin/
