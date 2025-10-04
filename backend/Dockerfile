FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod --mount=type=cache,target=/root/.cache/go-build \
    cd /app && go build -o /out/simstack ./cmd/server

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /out/simstack /usr/local/bin/simstack
EXPOSE 8080
ENV SIMSTACK_ADDR=:8080
CMD ["simstack"]


