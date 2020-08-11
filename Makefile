
install:
	@echo "Install dependencies"
	go get -d ./...

lint: install
	@echo "Running lint"
	go vet ./...
	gocritic check -enableAll ./...
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
