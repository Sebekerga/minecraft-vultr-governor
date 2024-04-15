
FROM golang:1.22.0

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY ./ ./

RUN go build -o /mc-vultr-gov ./run
RUN chmod +x /mc-vultr-gov
RUN touch /.env
CMD ["/mc-vultr-gov"]