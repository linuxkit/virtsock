.PHONY: all

DEPS=hvecho.c compat.h

all: Dockerfile $(DEPS)
	docker build -t hvecho:build .
	docker run --rm hvecho:build cat hvecho > hvecho
	chmod 755 hvecho

hvecho: $(DEPS)
	gcc -Wall -Werror -o hvecho hvecho.c

clean:
	rm -f hvecho
	docker images -q hvecho:build | xargs docker rmi -f
