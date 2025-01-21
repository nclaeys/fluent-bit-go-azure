CONTAINER_REGISTRY ?= nilli9990
docker_repo := $(CONTAINER_REGISTRY)/fluentbit-go-azure-logs-ingestion

artifact:
	go build -buildmode=c-shared -o out_azure_logs_ingestion.so

docker-push:
	docker buildx build --platform linux/amd64 . -t "$(docker_repo):latest" --push

clean:
	rm -rf *.so *.h *~
