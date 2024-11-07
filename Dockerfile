FROM golang:1.22-bullseye AS build
RUN apt-get update && apt-get install -y make gcc libc6-dev libsqlite3-dev wget libgcc1

WORKDIR /solver

RUN go env -w GOMODCACHE=/root/.cache/go-build

COPY go.mod go.sum Makefile ./
RUN --mount=type=cache,target=/root/.cache/go-build make deps

COPY . .

RUN --mount=type=cache,target=/root/.cache/go-build go build -tags "sqlite_omit_load_extension,linux,musl" -o build/skip_go_fast_solver ./cmd/solver

RUN wget -P /lib https://github.com/CosmWasm/wasmvm/releases/download/v2.1.0/libwasmvm.aarch64.so && \
    wget -P /lib https://github.com/CosmWasm/wasmvm/releases/download/v2.1.0/libwasmvm.x86_64.so

RUN cp /lib/x86_64-linux-gnu/libgcc_s.so.1 /lib/ || cp /lib/aarch64-linux-gnu/libgcc_s.so.1 /lib/

FROM gcr.io/distroless/base-debian11:debug
WORKDIR /solver
COPY --from=build /solver/build/skip_go_fast_solver /usr/local/bin/solver
COPY --from=build /solver/config /solver/config
COPY --from=build /solver/db/migrations /solver/db/migrations
COPY --from=build /lib/libwasmvm.* /lib/
COPY --from=build /lib/libgcc_s.so.1 /lib/

ENTRYPOINT ["solver", "quickstart=true"]
