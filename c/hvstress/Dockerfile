FROM alpine:3.3

RUN apk update && apk upgrade && apk add alpine-sdk util-linux-dev

RUN mkdir -p /hvstress
WORKDIR /hvstress

COPY . /hvstress

RUN make hvstress
