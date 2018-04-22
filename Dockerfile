FROM golang:1.10 as build-deps

COPY . /sower

WORKDIR /sower
ENV GOPATH=/sower

RUN echo "package handlers\n\nconst (" >src/handlers/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >>src/handlers/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >>src/handlers/gitversion.go \
    && echo ")" >src/handlers/gitversion.go \
    && cat src/handlers/gitversion.go

RUN CGO_ENABLED=0 GOOS=linux go build



FROM scratch
COPY --from=build-deps /sower/sower /sower
CMD ["/sower"]
