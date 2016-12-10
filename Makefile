.PHONY: build-in-container build-binaries hvgostress hvgoecho clean
DEPS:=$(wildcard pkg/*.go) $(wildcard examples/*.go) Dockerfile.build Makefile

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t virtsock-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}/build:/go/src/github.com/rneugeba/virtsock/build \
		virtsock-build

build-binaries: hvgostress hvgoecho
hvgostress: build/hvgostress.darwin build/hvgostress.linux build/hvgostress.exe
hvgoecho: build/hvgoecho.darwin build/hvgoecho.linux build/hvgoecho.exe


build/hvgostress.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/hvgostress.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go

build/hvgostress.darwin: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/hvgostress.go examples/common_vsock.go examples/common_darwin.go

build/hvgostress.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ examples/hvgostress.go examples/common_hvsock.go examples/common_windows.go



build/hvgoecho.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/hvgoecho.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go

build/hvgoecho.darwin: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/hvgoecho.go examples/common_vsock.go examples/common_darwin.go

build/hvgoecho.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ examples/hvgoecho.go examples/common_hvsock.go examples/common_windows.go


clean:
	rm -rf build

fmt:
	@echo "+ $@"
	@gofmt -s -l . 2>&1 | grep -v ^vendor/ | xargs gofmt -s -l -w

lint:
	@echo "+ $@"
	$(if $(shell which golint || echo ''), , \
		$(error Please install golint))
	@test -z "$$(golint ./... 2>&1 | grep -v ^vendor/ | grep -v mock/ | tee /dev/stderr)"
