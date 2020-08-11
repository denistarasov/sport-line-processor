
lint:

tests:
	@echo "Running tests"
	go test -v -race ./...

run:
	@echo "Starting server"
	sudo docker-compose up --build

stop:
	@echo "Starting server"
	sudo docker-compose stop
