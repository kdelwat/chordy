sources := $(wildcard *.go)

build: $(sources)
	go build -o chordy $(sources)
