docker_repo := nilli9990/fluentbit-go-azure-logs-ingestion

artifact:
	go build -buildmode=c-shared -o out_azurelogsingestion.so ./out_azurelogsingestion

docker-push:
	docker buildx build --platform linux/amd64 . -t "$(docker_repo):latest" --push

clean:
	rm -rf *.so *.h *~
