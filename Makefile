lint:
	docker run -it --rm -v $$(pwd):/app -w /app golangci/golangci-lint:v1.40.1 golangci-lint run -v