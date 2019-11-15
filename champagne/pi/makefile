.PHONY: all

all: pi.tar.gz

pi.tar.gz:
	@env GOOS=linux GOARCH=arm go build -o pi *.go
	tar -czvf pi.tar.gz pi templates public
	rm -f pi
