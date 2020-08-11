FROM golang:latest
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN go build -o main .
EXPOSE 8090
EXPOSE 8091
CMD ["/app/main", "-http", ":8090", "-grpc", ":8091", "-provider", "http://lines-provider:8000/api/v1/lines/", "-log", "info"]