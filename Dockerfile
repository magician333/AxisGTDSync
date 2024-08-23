FROM golang:1.23.0-alpine

WORKDIR /AxisGTDSync

COPY . .

RUN go mod download

RUN go build -o main .

EXPOSE 8080

CMD [ "./main" ]