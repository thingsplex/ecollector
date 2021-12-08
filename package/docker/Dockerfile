FROM alpine:edge

ARG TARGETARCH

COPY thingsplex/ /thingsplex/
COPY build/$TARGETARCH/ecollector /thingsplex/ecollector/

WORKDIR /thingsplex/ecollector/
CMD ["./ecollector"]
