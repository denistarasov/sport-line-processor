
install:
	@echo "Install dependencies"
	go get ./...

lint: install
	@echo "Running lint"
	go vet ./...
	GO111MODULE=on go get -v -u github.com/go-critic/go-critic/cmd/gocritic
	gocritic check -enableAll ./...
	go get golang.org/x/lint/golint
	golint ./...

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
