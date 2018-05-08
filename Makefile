all: transmission-filter

prepare:
	go get

transmission-filter: $(wildcard *.go)
	go build -o $@

clean:
	rm -f transmission-filter
