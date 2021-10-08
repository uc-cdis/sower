FROM quay.io/cdis/golang:1.17-bullseye as build-deps

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR $GOPATH/src/github.com/uc-cdis/sower/

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN COMMIT=$(git rev-parse HEAD); \
    VERSION=$(git describe --always --tags); \
    printf '%s\n' 'package handlers'\
    ''\
    'const ('\
    '    gitcommit="'"${COMMIT}"'"'\
    '    gitversion="'"${VERSION}"'"'\
    ')' > handlers/gitversion.go \
    && go build -o /sower

FROM scratch
COPY --from=build-deps /sower /sower
CMD ["/sower"]
