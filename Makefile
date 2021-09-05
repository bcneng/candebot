lint:
	golangci-lint run

clean:
    # '-f' to ignore when 'candebot' is not found
	rm -f candebot 

build: clean
	go build -v .
	@echo candebot built and ready to serve and protect.

test:
	go test -v ./...
