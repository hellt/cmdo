lint:
	docker run -it --rm -v $$(pwd):/work ghcr.io/hellt/golines:0.8.0 golines -w .
	docker run -it --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.60.3 golangci-lint run -v