FROM golang:1.21.4
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/mmf
RUN go build -o mmf .

CMD ["./mmf"]

