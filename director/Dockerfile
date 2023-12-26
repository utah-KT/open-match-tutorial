FROM golang:1.21.4
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/director
RUN go build -o director .

CMD ["./director"]
