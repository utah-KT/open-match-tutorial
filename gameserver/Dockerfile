FROM golang:1.21.4 as builder
WORKDIR /workspace

COPY go.mod go.sum /workspace/
COPY config/ /workspace/config/
COPY grpc/ /workspace/grpc/
RUN go mod download
COPY gameserver /workspace/gameserver
WORKDIR /workspace/gameserver
RUN CGO_ENABLED=0 GOOS=linux go build -o gameserver .

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /

COPY --from=builder /workspace/gameserver/gameserver /gameserver

USER nonroot:nonroot
CMD ["./gameserver"]
