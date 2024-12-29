TEMPDIR ?= .tmp

$(TEMPDIR):
	mkdir -p $(TEMPDIR)

build: $(TEMPDIR)
	cd cmd/runner && go build -o ../../$(TEMPDIR)/build/agent

.PHONY: test
test:
	go test ./... -v -parallel 4 -race
