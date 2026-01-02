FROM alpine:3.21

ARG TARGETPLATFORM

RUN apk add --no-cache ca-certificates tzdata

COPY ${TARGETPLATFORM}/flecto-manager /usr/local/bin/flecto-manager
COPY docker-entrypoint /usr/local/bin/docker-entrypoint

RUN chmod +x /usr/local/bin/flecto-manager /usr/local/bin/docker-entrypoint

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/docker-entrypoint"]
