FROM golang:latest

ARG HEIMDALL_DIR=/var/lib/heimdall
ENV HEIMDALL_DIR=$HEIMDALL_DIR

RUN apt-get update -y && apt-get upgrade -y \
    && apt install build-essential git -y \
    && mkdir -p $HEIMDALL_DIR

WORKDIR ${HEIMDALL_DIR}

# Copy go.mod and go.sum and download first to leverage Docker's cache
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN make build && cp build/heimdalld /usr/local/bin/

COPY docker/entrypoint.sh /usr/local/bin/entrypoint.sh

ENV SHELL /bin/bash
EXPOSE 1317 26656 26657

ENTRYPOINT ["entrypoint.sh"]
