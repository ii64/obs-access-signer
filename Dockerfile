FROM ii64/golang-zig:go1.18-alpine3.15-zig AS builder

WORKDIR /build
COPY . /build

RUN apk add --no-cache \
    make

RUN --mount=type=cache,mode=0755,target=/go/pkg/mod make dep
RUN make build


FROM gcr.io/distroless/static-debian11

WORKDIR /app
COPY --from=builder /build/obs-access-signer /app/obs-access-signer

ENTRYPOINT [ "/app/obs-access-signer" ]