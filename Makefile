run:
	go run ./main.go

build-project:
	mkdir -p build
	go build -o build/scanner 

run-build:
	./build/scanner
