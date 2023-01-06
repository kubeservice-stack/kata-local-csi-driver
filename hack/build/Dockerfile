# Build the manager binary
FROM golang:1.18.5-alpine as builder

MAINTAINER dongjiang1989@126.com
LABEL blog="https://stack.kubeservice.cn"

RUN apk add --no-cache gcc musl-dev libc6-compat

WORKDIR /workspace
COPY go.mod go.mod
COPY go.sum go.sum

# Copy the go source
COPY cmd/ cmd/
COPY pkg/ pkg/
COPY hack/ hack/
COPY vendor/ vendor/

# package image
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-linkmode external -extldflags -static" -o kata-csi-driver ./cmd/driver/main.go

FROM ubuntu:20.04

MAINTAINER dongjiang1989@126.com
LABEL blog="https://stack.kubeservice.cn"

RUN apt update && apt upgrade -y && apt install -y ca-certificates file tzdata lvm2

COPY --from=builder /workspace/kata-csi-driver /bin/kata-csi-driver
COPY --from=builder /workspace/hack/build/entrypoint.sh /entrypoint.sh
COPY --from=builder /workspace/hack/build/nsenter /nsenter

RUN chmod +x /bin/kata-csi-driver && chmod +x /entrypoint.sh && sed -i 's/use_lvmetad\ =\ 1/use_lvmetad\ =\ 0/g' /etc/lvm/lvm.conf

ENTRYPOINT ["/entrypoint.sh"]
