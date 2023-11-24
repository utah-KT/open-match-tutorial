FROM golang:alpine
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/mmf
RUN go mod tidy
RUN go build -o mmf .

CMD ["./mmf"]

