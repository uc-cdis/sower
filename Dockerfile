FROM golang:1.12-stretch as build-deps

WORKDIR $GOPATH/src/github.com/uc-cdis/sower/
COPY . ./

# Populate git version info into the code
RUN echo "package handlers\n\nconst (" > handlers/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >> handlers/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >> handlers/gitversion.go \
    && echo ")" >> handlers/gitversion.go

ENV GO111MODULE=on
# go get and build
RUN go get -d -v
RUN go build -o /sower

# Store only the resulting binary in the final image
# Resulting in significantly smaller docker image size
FROM scratch
COPY --from=build-deps /sower /sower
CMD ["/sower"]
