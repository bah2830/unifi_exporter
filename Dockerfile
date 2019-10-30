FROM golang:latest

# ENV GO111MODULE=on

WORKDIR /app
COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o unifi_exporter ./cmd/unifi_exporter/main.go

EXPOSE 9130
ENTRYPOINT ["/app/unifi_exporter"]
