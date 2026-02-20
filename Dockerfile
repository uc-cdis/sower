FROM quay.io/cdis/golang-build-base:go1.26.0 AS build-deps

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

WORKDIR $GOPATH/src/github.com/uc-cdis/sower/

COPY --chown=1000:1000 go.mod .
COPY --chown=1000:1000 go.sum .

RUN go mod download

COPY --chown=1000:1000 . .

RUN GITCOMMIT=$(git rev-parse HEAD) \
    GITVERSION=$(git describe --always --tags) \
    && go build \
    -ldflags="-X 'github.com/uc-cdis/sower/handlers/version.GitCommit=${GITCOMMIT}' -X 'github.com/uc-cdis/sower/handlers/version.GitVersion=${GITVERSION}'" \
    -o /go/bin/sower

USER root
RUN echo "nobody:x:65534:65534:Nobody:/:" > /etc_passwd
USER gen3

FROM scratch
COPY --from=build-deps /etc_passwd /etc/passwd
COPY --from=build-deps /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build-deps /go/bin/sower /sower
USER nobody
CMD ["/sower"]
