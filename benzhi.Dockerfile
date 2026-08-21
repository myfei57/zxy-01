FROM golang:1.23

WORKDIR /app

COPY go.mod go.sum ./
COPY vendor ./vendor
COPY . .

RUN go build -mod=vendor ./... \
    && go vet -mod=vendor ./... \
    && go test -mod=vendor -count=1 ./...

ENV GOPROXY=off \
    GOSUMDB=off

CMD ["bash"]
