.DEFAULT_GOAL := test

ifndef TEST_RESULTS
	TEST_RESULTS := 'target'
endif

.PHONY: test build all

all: test build install

dpes:
    dep ensure

test-report-dir:
	mkdir -p ${TEST_RESULTS}

test: test-report-dir
	go test \
		-race -v  \
		./...
build:
	go build

install:
	go install
