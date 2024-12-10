.PHONY: all clean install

all: build

build:
	go build ./cmd/cometbftsignrate

install:
	go install ./cmd/cometbftsignrate

clean:
	rm -f ./cometbftsignrate