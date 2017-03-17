FROM golang:1.8

ARG GOBINDATAVERSION=a0ff2567cfb70903282db057e799fd826784d41d

RUN git clone https://github.com/jteeuwen/go-bindata.git $GOPATH/src/github.com/jteeuwen/go-bindata
WORKDIR $GOPATH/src/github.com/jteeuwen/go-bindata
RUN git checkout $GOBINDATAVERSION
RUN go get github.com/jteeuwen/go-bindata/...

ENV CGO_ENABLED 0
WORKDIR /go/src/github.com/gowncloud/gowncloud

EXPOSE 8080

ENTRYPOINT go generate && go build && ./gowncloud -d
