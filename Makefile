.PHONY: build-in-container build-binaries virtsock_stress clean
DEPS:=$(wildcard pkg/*.go) $(wildcard cmd/virtsock_stress/*.go) $(wildcard cmd/vsudd/*.go) Dockerfile.build Makefile

build-in-container: $(DEPS) clean
	@echo "+ $@"
	@docker build -t virtsock-build -f ./Dockerfile.build .
	@docker run --rm \
		-v ${CURDIR}/bin:/go/src/github.com/linuxkit/virtsock/bin \
		virtsock-build

build-binaries: vsudd virtsock_stress
virtsock_stress: bin/virtsock_stress.darwin bin/virtsock_stress.linux bin/virtsock_stress.exe
vsudd: bin/vsudd.linux 

bin/vsudd.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ -buildmode pie --ldflags '-s -w -extldflags "-static"' \
		cmd/vsudd/main.go cmd/vsudd/vsyslog.go

bin/virtsock_stress.linux: $(DEPS)
	@echo "+ $@"
	GOOS=linux GOARCH=amd64 \
	go build -o $@ -buildmode pie --ldflags '-s -w -extldflags "-static"' \
		cmd/virtsock_stress/virtsock_stress.go cmd/virtsock_stress/common_hvsock.go cmd/virtsock_stress/common_vsock.go cmd/virtsock_stress/common_linux.go

bin/virtsock_stress.darwin: $(DEPS)
	@echo "+ $@"
	GOOS=darwin GOARCH=amd64 \
	go build -o $@ --ldflags '-extldflags "-fno-PIC"' \
		cmd/virtsock_stress/virtsock_stress.go cmd/virtsock_stress/common_vsock.go cmd/virtsock_stress/common_darwin.go

bin/virtsock_stress.exe: $(DEPS)
	@echo "+ $@"
	GOOS=windows GOARCH=amd64 \
	go build -o $@ cmd/virtsock_stress/virtsock_stress.go cmd/virtsock_stress/common_hvsock.go cmd/virtsock_stress/common_windows.go

# Target to build a bootable EFI ISO and kernel+initrd
linuxkit: build-in-container Dockerfile.linuxkit hvtest.yml
	$(MAKE) -C c build-in-container
	docker build -t hvtest-local -f Dockerfile.linuxkit .
	moby build -output kernel+initrd,iso-efi hvtest.yml

clean:
	rm -rf bin c/build

fmt:
	@echo "+ $@"
	@gofmt -s -l . 2>&1 | grep -v ^vendor/ | xargs gofmt -s -l -w

lint:
	@echo "+ $@"
	$(if $(shell which golint || echo ''), , \
		$(error Please install golint))
	@test -z "$$(golint ./... 2>&1 | grep -v ^vendor/ | grep -v mock/ | tee /dev/stderr)"
