FROM golang:1.23-bookworm AS builder
WORKDIR /build
COPY go.mod main.go ./
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o server .

FROM texlive/texlive:latest
COPY --from=builder /build/server /usr/local/bin/server
COPY entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh
EXPOSE 80
ENTRYPOINT ["/entrypoint.sh"]
