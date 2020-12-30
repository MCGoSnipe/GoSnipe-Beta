env GOOS=linux GOARCH=amd64 go build -o bin/linux snipe.go
env GOOS=darwin GOARCH=amd64 go build -o bin/mac snipe.go
env GOOS=windows GOARCH=amd64 go build -o bin/win snipe.go