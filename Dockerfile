FROM golang:1-alpine AS gobuild
ENV GOPROXY https://proxy.golang.org
WORKDIR /work
COPY . .
RUN apk -U add git build-base && make ratelimit-exporter

FROM alpine:latest
COPY --from=gobuild /work/ratelimit-exporter /usr/local/bin/ratelimit-exporter
# Git is needed because this'll get called as a CI job from Gitlab
ENTRYPOINT ["/usr/local/bin/ratelimit-exporter"]
