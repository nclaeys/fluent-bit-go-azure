docker_repo := nilli9990/fluentbit-go-azure-logs-ingestion
FLUENTBIT_VERSION := 1.9.10
PLUGIN_VERSION := 0.0.1

build:
	go build -buildmode=c-shared -o out_azurelogsingestion.so ./out_azurelogsingestion

docker-push:
	docker buildx build --platform linux/amd64 . -t "$(docker_repo):v$(FLUENTBIT_VERSION)-v$(PLUGIN_VERSION)" --push

docker-build:
	docker buildx build --platform linux/amd64 . -t "$(docker_repo):v$(FLUENTBIT_VERSION)-v$(PLUGIN_VERSION)"

clean:
	rm -rf *.so *.h *~

test:
	go test $(TEST_OPTS) ./out_azurelogsingestion