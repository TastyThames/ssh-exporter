# syntax=docker/dockerfile:1

FROM golang:1.22 AS build
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/ssh-exporter ./cmd/ssh-exporter

FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=build /out/ssh-exporter /app/ssh-exporter

EXPOSE 9222
USER nonroot:nonroot
ENTRYPOINT ["/app/ssh-exporter"]
