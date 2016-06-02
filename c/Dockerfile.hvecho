FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk util-linux-dev

RUN mkdir -p /hvecho
WORKDIR /hvecho

COPY . /hvecho

RUN make hvecho
