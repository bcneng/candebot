lint:
	go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.6.0 run

clean:
	@# '-f' to ignore when 'candebot' is not found
	rm -f candebot 

build: clean
	go build -v -ldflags "-X main.Version=$$(git rev-parse --short HEAD)" .
	@echo candebot built and ready to serve and protect.

test:
	go test -v ./...

run:
	go run .

simulator:
	open http://localhost:8080/_simulator/
