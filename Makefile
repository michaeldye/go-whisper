SHELL = /bin/bash -e

all: test

deep-clean:
	tools/deep-clean CLEAN

deps:
	go get

repo-fork-sync:
	tools/repo-fork-sync

# only unit tests
test:
	go test -v --cover

## both integration and unit tests, but in-container
#test-integration:
#	cd integration; docker-compose rm -fv;	docker-compose build --no-cache; 	docker-compose run whisper-test
#
.PHONY: deep-clean deps repo-fork-sync test test-integration
