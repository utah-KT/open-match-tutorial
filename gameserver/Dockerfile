FROM golang:alpine
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/gameserver
RUN go mod tidy
RUN go build -o gameserver .

CMD ["./gameserver"]

