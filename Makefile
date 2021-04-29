binary=rld

release:
	GOOS=linux GOARCH=amd64 go build -o ./bin/$(binary)-linux
	GOOS=darwin GOARCH=amd64 go build -o ./bin/$(binary)-darwin
	GOOS=windows GOARCH=amd64 go build -o ./bin/$(binary)-windows
