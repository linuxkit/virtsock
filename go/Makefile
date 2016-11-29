
.PHONY: all hvstress hvgoecho hvgostress hvgoecho

all: hvgostress hvgoecho

hvgostress: examples/hvgostress.darwin examples/hvgostress.linux examples/hvgostress.exe
hvgoecho: examples/hvgoecho.darwin examples/hvgoecho.linux examples/hvgoecho.exe


DEPS:=$(wildcard hvsock/*.go)  Dockerfile Makefile

examples/hvgostress.linux: examples/hvgostress.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=linux -e GOARCH=amd64 -t hvsock:build \
		go build -o hvgostress.linux --ldflags '-extldflags "-fno-PIC"' hvgostress.go common_hvsock.go common_vsock.go common_linux.go


examples/hvgostress.darwin: examples/hvgostress.go examples/common_vsock.go examples/common_darwin.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=darwin -e GOARCH=amd64 -t hvsock:build \
		go build -o hvgostress.darwin  --ldflags '-extldflags "-fno-PIC"' hvgostress.go common_vsock.go common_darwin.go

examples/hvgostress.exe: examples/hvgostress.go examples/common_hvsock.go examples/common_windows.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=windows -e GOARCH=amd64 -t hvsock:build \
		go build hvgostress.go common_hvsock.go common_windows.go

examples/hvgoecho.linux: examples/hvgoecho.go examples/common_hvsock.go examples/common_vsock.go examples/common_linux.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=linux -e GOARCH=amd64 -t hvsock:build \
		go build -o hvgoecho.linux  --ldflags '-extldflags "-fno-PIC"' hvgoecho.go common_hvsock.go common_vsock.go common_linux.go

examples/hvgoecho.darwin: examples/hvgoecho.go examples/common_vsock.go examples/common_darwin.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=darwin -e GOARCH=amd64 -t hvsock:build \
		go build -o hvgoecho.darwin  --ldflags '-extldflags "-fno-PIC"' hvgoecho.go common_vsock.go common_darwin.go

examples/hvgoecho.exe: examples/hvgoecho.go examples/common_hvsock.go examples/common_windows.go $(DEPS)
	docker build -t hvsock:build .
	docker run -v $(PWD):/go -e GOOS=windows -e GOARCH=amd64 -t hvsock:build \
		go build hvgoecho.go common_hvsock.go common_windows.go

clean:
	rm -f examples/hvgostress.darwin examples/hvgostress.linux examples/hvgostress.exe
	rm -f examples/hvgoecho.darwin examples/hvgoecho.linux examples/hvgoecho.exe
