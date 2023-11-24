FROM golang:alpine
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/front
RUN go mod tidy
RUN go build -o front .

CMD ["./front"]

