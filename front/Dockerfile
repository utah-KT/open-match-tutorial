FROM golang:1.21.4 as builder
WORKDIR /workspace

COPY go.mod go.sum /workspace/
COPY config/ /workspace/config/
COPY grpc/ /workspace/grpc/
RUN go mod download
COPY front /workspace/front
WORKDIR /workspace/front
RUN CGO_ENABLED=0 GOOS=linux go build -o front .

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /

COPY --from=builder /workspace/front/front /front

USER nonroot:nonroot
CMD ["./front"]
