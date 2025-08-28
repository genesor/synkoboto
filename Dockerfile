# build stage
FROM golang:1.25-alpine AS build

RUN apk update && apk --no-cache add git ca-certificates musl-dev && update-ca-certificates && rm -rf /var/cache/apk/*

WORKDIR /app

COPY . .
RUN GOFLAGS='-mod=vendor' go build -o synkoboto ./pkg/cmd

# release stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates && update-ca-certificates

EXPOSE 80

WORKDIR /app
COPY --from=build /app/synkoboto /app/

ENTRYPOINT ./synkoboto
