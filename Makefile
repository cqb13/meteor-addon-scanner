run:
	go run ./scanner.go

build:
	mkdir -p build
	go build -o build/scanner ./src

run-build:
	./build/scanner
