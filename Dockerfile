FROM c.rzp.io/proxy_dockerhub/library/golang:1.17-alpine3.13 as builder

ADD . /src

WORKDIR /src

COPY go.mod .

RUN go build main.go

CMD /src/main