FROM golang:alpine
WORKDIR /workspace

COPY . /workspace
WORKDIR /workspace/director
RUN go mod tidy
RUN go build -o director .

CMD ["./director"]

