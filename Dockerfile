FROM golang as build

WORKDIR /go/src/app
COPY . .

RUN go get -d -v ./...
RUN GO111MODULE=off CGO_ENABLED=0 go build -o /app -trimpath -tags netgo -ldflags '-s -w'

FROM scratch

COPY --from=build /app /app

ENTRYPOINT ["/app"]
