CONTAINER_REGISTRY ?= nilli9990
docker_repo := $(CONTAINER_REGISTRY)/fluent-bit-go-azure

artifact:
	go build -buildmode=c-shared -o out_azure_log_ingestion_workload.so

docker-push:
	docker buildx build --platform linux/amd64 . -t "$(docker_repo):latest" --push

clean:
	rm -rf *.so *.h *~
