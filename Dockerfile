# ─── BUILDER STAGE ───────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

ARG HEIMDALL_DIR=/var/lib/heimdall/
ENV HEIMDALL_DIR=${HEIMDALL_DIR}

RUN apk add --no-cache build-base git linux-headers

WORKDIR ${HEIMDALL_DIR}

COPY go.mod go.sum ./

RUN --mount=type=ssh \
    --mount=type=cache,target=/go/pkg/mod \
    go mod download

COPY . .

RUN --mount=type=ssh \
    --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    make build

# ─── RUNTIME STAGE ────────────────────────────────────────────────────────────────
FROM alpine:latest

ARG HEIMDALL_DIR=/var/lib/heimdall/
ENV HEIMDALL_DIR=${HEIMDALL_DIR}

RUN apk add --no-cache bash ca-certificates && \
    mkdir -p ${HEIMDALL_DIR}

WORKDIR ${HEIMDALL_DIR}

COPY --from=builder ${HEIMDALL_DIR}/build/heimdalld /usr/local/bin/heimdalld

EXPOSE 1317 26656 26657

ENTRYPOINT ["/usr/local/bin/heimdalld"]
