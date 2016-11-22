DEPS=compat.h Makefile
MSBUILD="/c/Program Files (x86)/MSBuild/14.0/Bin/MSBuild.exe"

.PHONY: all alpine hvecho.alpine hvstress.alpine clean

all: alpine windows

alpine: hvecho.alpine hvstress.alpine hvbench.alpine
windows: hvecho.exe hvstress.exe hvbench.exe


hvecho.alpine: hvecho.c Dockerfile.hvecho $(DEPS)
	docker build -f Dockerfile.hvecho -t hvecho:build .
	docker run --rm hvecho:build cat hvecho > hvecho
	chmod 755 hvecho

hvecho: hvecho.c $(DEPS)
	gcc -Wall -Werror -o hvecho hvecho.c

hvecho.exe: hvecho.c hvecho.vcxproj hvecho.sln $(DEPS)
	$(MSBUILD) hvecho.sln
	cp x64/Debug/hvecho.exe .


hvstress.alpine: hvstress.c Dockerfile.hvstress $(DEPS)
	docker build -f Dockerfile.hvstress -t hvstress:build .
	docker run --rm hvstress:build cat hvstress > hvstress
	chmod 755 hvstress

hvstress: hvstress.c $(DEPS)
	gcc -Wall -Werror -o hvstress hvstress.c

hvstress.exe: hvstress.c hvstress.vcxproj hvstress.sln $(DEPS)
	$(MSBUILD) hvstress.sln
	cp x64/Debug/hvstress.exe .


hvbench.alpine: hvbench.c Dockerfile.hvbench $(DEPS)
	docker build -f Dockerfile.hvbench -t hvbench:build .
	docker run --rm hvbench:build cat hvbench > hvbench
	chmod 755 hvbench

hvbench: hvbench.c $(DEPS)
	gcc -Wall -Werror -o hvbench hvbench.c

hvbench.exe: hvbench.c hvbench.vcxproj hvbench.sln $(DEPS)
	$(MSBUILD) hvbench.sln
	cp x64/Debug/hvbench.exe .


clean:
	rm -f hvecho hvecho.exe
	rm -f hvstress hvstress.exe
	rm -f hvbench hvbench.exe
	rm -rf Debug x64
	docker images -q hvecho:build | xargs docker rmi -f
	docker images -q hvstress:build | xargs docker rmi -f
	docker images -q hvbench:build | xargs docker rmi -f
