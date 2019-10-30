.PHONY: all build docker

build:
	go build -mod=vendor ./cmd/unifi_exporter

docker:
	docker build -t unifi_exporter .
