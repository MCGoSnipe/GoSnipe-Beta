set GOOS=linux&set GOARCH=amd64&go build -o bin/linux snipe.go
set GOOS=darwin&set  GOARCH=amd64&go build -o bin/mac snipe.go
set GOOS=windows&set  GOARCH=amd64&go build -o bin/win snipe.go