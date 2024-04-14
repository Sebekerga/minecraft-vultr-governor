
FROM golang:1.22.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o /mc-vultr-gov
RUN chmod +x /mc-vultr-gov
RUN touch /.env
CMD ["/mc-vultr-gov"]