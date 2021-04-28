FROM quay.io/cdis/golang:1.12-stretch as build-deps

WORKDIR $GOPATH/src/github.com/uc-cdis/sower/

COPY go.mod .
COPY go.sum .

ENV GO111MODULE=on
RUN go mod download

COPY . .

# Populate git version info into the code
RUN echo "package handlers\n\nconst (" > handlers/gitversion.go \
    && COMMIT=`git rev-parse HEAD` && echo "    gitcommit=\"${COMMIT}\"" >> handlers/gitversion.go \
    && VERSION=`git describe --always --tags` && echo "    gitversion=\"${VERSION}\"" >> handlers/gitversion.go \
    && echo ")" >> handlers/gitversion.go

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -o /sower

# Store only the resulting binary in the final image
# Resulting in significantly smaller docker image size
FROM scratch
COPY --from=build-deps /sower /sower
CMD ["/sower"]
