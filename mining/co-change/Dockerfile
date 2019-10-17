FROM golang:1.11-alpine

WORKDIR /app

RUN apk add --no-cache git

ADD go.mod .
ADD go.sum .

RUN go mod download

ADD . .

RUN go build -o app

FROM alpine

RUN apk add --no-cache git

ADD run.sh /
COPY --from=0 /app/app /co-change

RUN chmod +x /run.sh

ENTRYPOINT ["/run.sh"]
