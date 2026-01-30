FROM golang:1.25.4 
WORKDIR /build
COPY go.mod ./
RUN go mod download && go mod verify
COPY . .
RUN go build \
    -o /exchange \
    ./cmd/exchange/
CMD ["/exchange"]
