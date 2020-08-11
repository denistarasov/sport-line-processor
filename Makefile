
install:
	@echo "Install dependencies"
	go get ./...

lint: install
	@echo "Running lint"
	go vet ./...
    curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.30.0
    $(go env GOPATH)/bin/golangci-lint run -E gocritic -E golint

tests:
	@echo "Running tests"
	go test -v -race ./...

run:
	@echo "Starting server"
	sudo docker-compose up --build

stop:
	@echo "Starting server"
	sudo docker-compose stop

build:
	@echo "Building project"
	go build -o main .
