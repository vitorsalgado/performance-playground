FROM golang:1.25.4 AS builder
WORKDIR /src

ARG TARGETOS=linux
ARG TARGETARCH=amd64
ARG APP=exchange

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY internal ./internal
COPY $APP ./$APP

RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
  go build \
  -trimpath \
  -buildvcs=false \
  -ldflags="-s -w" \
  -o /out/app \
  ./$APP

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /out/app /app
EXPOSE 8080
ENTRYPOINT ["/app"]
