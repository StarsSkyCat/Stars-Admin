FROM golang:1.27.4-bookworm@sha256:4fa7c968ba93f1635670522dbb01a2c78150443e1385746924566079f36570ea AS builder
ENV GOTOOLCHAIN=local
WORKDIR /src
RUN go install github.com/swaggo/swag/cmd/swag@v1.16.6
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN make build && mkdir -p /out/logs /out/uploads /out/tmp && cp bin/stars-admin /out/stars-admin && cp -r configs /out/configs
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/probe ./cmd/probe

# 静态二进制使用非 root 用户运行，保留 TLS 根证书。
FROM scratch
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --chown=65532:65532 --from=builder /out/ /app/
WORKDIR /app
ENV APP_ENV=prod TMPDIR=/app/tmp
USER 65532:65532
EXPOSE 8080
STOPSIGNAL SIGTERM
ENTRYPOINT ["/app/stars-admin"]
CMD ["serve"]
