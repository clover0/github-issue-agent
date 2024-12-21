TEMPDIR ?= .tmp

$(TEMPDIR):
	mkdir -p $(TEMPDIR)

build: $(TEMPDIR)
	cd cmd/runner && go build -o ../../$(TEMPDIR)/build/agent

test:
	go test ./... -v
