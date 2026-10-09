BIN := threads-leads.exe

.PHONY: build test vet run login explore tidy

build:
	go build -o $(BIN) ./cmd/app

test:
	go test ./...

vet:
	go vet ./...

tidy:
	go mod tidy

run: build
	./$(BIN) run $(ARGS)

login: build
	./$(BIN) login

explore: build
	./$(BIN) explore "$(Q)"
