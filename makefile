build:
	go build -o bin/notectl src/notectl/notectl.go

clean:
	rm -rf bin/
	mkdir bin

install:
	go install ./build/notectl

compile:
	echo "Compiling for each supported platform..."
	GOOS=linux GOARCH=amd64 go build -o bin/notectl-linux-x86_64 src/notectl/notectl.go
	GOOS=darwin GOARCH=amd64 go build -o bin/notectl-darwin-x86_64 src/notectl/notectl.go
	GOOS=windows GOARCH=amd64 go build -o bin/notectl-windows-x86_64.exe src/notectl/notectl.go

fill:
	bin/notectl new "Note1"
	bin/notectl new "Note2"
	bin/notectl new "Note3"
	bin/notectl new "Note4"
	bin/notectl new "Note5"
	bin/notectl new "Note6"
	bin/notectl new "Note7"
	bin/notectl new "Note8"