FROM golang:alpine

WORKDIR /go/src/app

COPY . .

EXPOSE 8081

RUN go build -o main .

CMD ["./main"]
