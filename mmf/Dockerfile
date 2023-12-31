FROM golang:1.21.4 as builder
WORKDIR /workspace

COPY go.mod go.sum /workspace/
COPY config/ /workspace/config/
RUN go mod download
COPY mmf /workspace/mmf
WORKDIR /workspace/mmf
RUN CGO_ENABLED=0 GOOS=linux go build -o mmf .

FROM gcr.io/distroless/static-debian11:nonroot
WORKDIR /

COPY --from=builder /workspace/mmf/mmf /mmf

USER nonroot:nonroot
CMD ["./mmf"]
