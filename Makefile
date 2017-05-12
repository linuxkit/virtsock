.PHONY: build-in-container build-binaries virtsock_stress virtsock_echo clean
DEPS:=$(wildcard pkg/*.go) $(wildcard examples/*.go) Dockerfile.build Makefile

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t virtsock-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}/build:/go/src/github.com/linuxkit/virtsock/build \
		virtsock-build

build-binaries: virtsock_stress virtsock_echo
virtsock_stress: build/virtsock_stress.darwin build/virtsock_stress.linux build/virtsock_stress.exe
virtsock_echo: build/virtsock_echo.darwin build/virtsock_echo.linux build/virtsock_echo.exe


build/virtsock_stress.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ -buildmode pie --ldflags '-s -w -extldflags "-static"' \
		examples/virtsock_stress.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go

build/virtsock_stress.darwin: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/virtsock_stress.go examples/common_vsock.go examples/common_darwin.go

build/virtsock_stress.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ examples/virtsock_stress.go examples/common_hvsock.go examples/common_windows.go



build/virtsock_echo.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ -buildmode pie --ldflags '-s -w -extldflags "-static"' \
		examples/virtsock_echo.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go

build/virtsock_echo.darwin: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		examples/virtsock_echo.go examples/common_vsock.go examples/common_darwin.go

build/virtsock_echo.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ examples/virtsock_echo.go examples/common_hvsock.go examples/common_windows.go


# Target to build a bootable EFI ISO
linuxkit: hvtest-efi.iso
hvtest-efi.iso: build-in-container Dockerfile.linuxkit hvtest.yml
	$(MAKE) -C c build-in-container
	docker build -t hvtest-local -f Dockerfile.linuxkit .
	moby build hvtest

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
