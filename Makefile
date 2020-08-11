
install:
	@echo "Install dependencies"
	export GOPATH=$HOME/golang
	export GOROOT=/usr/local/opt/go/libexec
	export PATH=$PATH:$GOPATH/bin
	export PATH=$PATH:$GOROOT/bin
	export CGO_ENABLED=1; export CC=gcc;
	export PATH=$PATH:/Users/james/golang/bin/golint
	export GO111MODULE=on
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
