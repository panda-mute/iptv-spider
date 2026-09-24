FROM golang:1.24-bookworm AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /iptv-spider .

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates ffmpeg tzdata && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /iptv-spider /usr/local/bin/iptv-spider
COPY config.yaml ./config.yaml
ENV TZ=Asia/Shanghai IPTV_DATA_DIR=/app/data
EXPOSE 8888
VOLUME ["/app/data"]
ENTRYPOINT ["iptv-spider"]
