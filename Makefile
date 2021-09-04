lint:
	golangci-lint run

clean:
    # '-f' to ignore when 'candebot' is not found
	rm -f candebot 

get-dependencies:
	echo "Getting dependencies..."
	go get -v -t -d ./...

fast-build:
	go build -v .

build: clean get-dependencies fast-build
	echo "'candebot' built and ready to serve and protect."

test: fast-build
	go test -v ./...