run:
	templ generate
	go run main.go

build:
	templ generate
	go build -o invoice-app main.go
