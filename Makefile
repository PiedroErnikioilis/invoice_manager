run:
	go tool templ generate
	go run main.go

build:
	go tool templ generate
	go build -o invoice-app main.go
