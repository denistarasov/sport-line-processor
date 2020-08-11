
install:
	@echo "Install dependencies"
	go get ./...

lint: install
	@echo "Running lint"
	go vet ./...
	go get golang.org/x/lint/golint
	golint ./...
	go get github.com/go-critic/go-critic/...
	gocritic check -enableAll ./...

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
