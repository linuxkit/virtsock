.PHONY: all

DEPS=hvstress.c compat.h

all: Dockerfile $(DEPS)
	docker build -t hvstress:build .
	docker run --rm hvstress:build cat hvstress > hvstress
	chmod 755 hvstress

hvstress: $(DEPS)
	gcc -Wall -Werror -o hvstress hvstress.c -lpthread

clean:
	rm -f hvstress
	docker images -q hvstress:build | xargs docker rmi -f
