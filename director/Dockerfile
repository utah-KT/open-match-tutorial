FROM golang:1.21.4 as builder
WORKDIR /workspace

COPY go.mod go.sum /workspace/
COPY config/ /workspace/config/
COPY grpc/ /workspace/grpc/
RUN go mod download
COPY director /workspace/director
WORKDIR /workspace/director
RUN CGO_ENABLED=0 GOOS=linux go build -o director .

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /

COPY --from=builder /workspace/director/director /director

USER nonroot:nonroot
CMD ["./director"]
