FROM golang as build

WORKDIR /go/src/app
COPY . .

RUN go mod tidy
RUN CGO_ENABLED=0 go build -o /app -trimpath -ldflags '-s -w'

FROM scratch

COPY --from=build /app /app

ENTRYPOINT ["/app"]
