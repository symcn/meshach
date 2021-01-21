FROM alpine:3.11.3

COPY ./bin/meshach .
ENTRYPOINT ["/meshach"]
