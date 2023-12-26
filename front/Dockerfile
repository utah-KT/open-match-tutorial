FROM golang:1.21.4
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/front
RUN go build -o front .

CMD ["./front"]

