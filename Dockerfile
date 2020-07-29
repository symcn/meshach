FROM alpine:3.11.3

COPY ./bin/mesh-operator .
ENTRYPOINT ["/mesh-operator"]
