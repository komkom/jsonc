FROM golang:1.16-alpine AS build

# get the latest security updates
# https://github.com/hadolint/hadolint/issues/562
RUN apk --no-cache upgrade

WORKDIR /go/src/app
COPY . .

# build a static binary for running from scratch
# https://github.com/jeremyhuiskamp/golang-docker-scratch
RUN go get -d -v ./...; \
CGO_ENABLED=0 go install -ldflags '-extldflags "-static"' -tags timetzdata -v ./...

FROM scratch
COPY --from=build /go/bin/cmd /bin/jsonc

CMD ["/bin/jsonc"]
