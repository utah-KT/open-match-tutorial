FROM golang:1.21.4
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/gameserver
RUN go build -o gameserver .

CMD ["./gameserver"]

