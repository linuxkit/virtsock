
.PHONY: all hvstress hvgoecho

all: hvgostress hvgoecho

hvgostress: examples/hvgostress examples/hvgostress.exe
hvgoecho: examples/hvgoecho examples/hvgoecho.exe


DEPS:=$(wildcard hvsock/*.go)  Dockerfile Makefile

examples/hvgostress: examples/hvgostress.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=linux -e GOARCH=amd64 -t hvsock:build \
		go build  --ldflags '-extldflags "-fno-PIC"' hvgostress.go

examples/hvgostress.exe: examples/hvgostress.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=windows -e GOARCH=amd64 -t hvsock:build \
		go build  hvgostress.go

examples/hvgoecho: examples/hvgoecho.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=linux -e GOARCH=amd64 -t hvsock:build \
		go build  --ldflags '-extldflags "-fno-PIC"' hvgoecho.go

examples/hvgoecho.exe: examples/hvgoecho.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=windows -e GOARCH=amd64 -t hvsock:build \
		go build  hvgoecho.go

clean:
	rm examples/hvgostress examples/hvgostress.exe
	rm examples/hvgoecho examples/hvgoecho.exe
