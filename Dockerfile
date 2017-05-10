FROM golang
LABEL maintainer "Peter Etelej <peter@etelej.com>"

RUN go get -u github.com/peteretelej/kvstore

EXPOSE 8080

CMD ["kvstore"]


