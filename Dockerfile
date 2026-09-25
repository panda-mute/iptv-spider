FROM --platform=$BUILDPLATFORM golang:1.24-alpine AS build
ARG TARGETOS TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -trimpath -ldflags="-s -w" -o /iptv-spider .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates ffmpeg tzdata
WORKDIR /app
COPY --from=build /iptv-spider /usr/local/bin/iptv-spider
COPY config.yaml ./config.yaml
ENV TZ=Asia/Shanghai IPTV_DATA_DIR=/app/data
EXPOSE 8888
VOLUME ["/app/data"]
ENTRYPOINT ["iptv-spider"]
