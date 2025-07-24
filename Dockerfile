# ─── BUILDER STAGE ───────────────────────────────────────────────────────────────
FROM golang:1.24-alpine AS builder

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
FROM alpine:3.22

ARG HEIMDALL_DIR=/var/lib/heimdall/
ENV HEIMDALL_DIR=${HEIMDALL_DIR}

RUN apk add --no-cache bash ca-certificates && \
    mkdir -p ${HEIMDALL_DIR}

WORKDIR ${HEIMDALL_DIR}

COPY --from=builder ${HEIMDALL_DIR}/build/heimdalld /usr/local/bin/heimdalld
COPY --from=builder ${HEIMDALL_DIR}/docker/entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 1317 26656 26657

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
