ARG ARCH=amd64

# Build stage
FROM golang:1.21-alpine3.17 AS builder
ENV GO111MODULE=on
ENV GOPROXY=https://goproxy.cn
WORKDIR /src
ADD . /src
RUN go mod tidy
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${ARCH} go build -ldflags='-s -w -extldflags "-static"' -o elector cmd/leader-elector/main.go

FROM gcr.io/distroless/static:nonroot-${ARCH}
USER root:root
WORKDIR /app
COPY --from=builder --chown=root:root /src/elector /app/
ENTRYPOINT ["./elector"]
