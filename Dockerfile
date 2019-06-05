FROM golang:1.12-alpine as builder
RUN apk add --no-cache git
ENV GOOS=linux
ENV CGO_ENABLED=0
ENV GO111MODULE=on

COPY . /src
WORKDIR /src

RUN rm -f go.sum
RUN go get
RUN go test ./...
RUN go build -a -installsuffix cgo -o mutatingflow

FROM alpine:3.9
RUN apk add --no-cache ca-certificates
WORKDIR /app
COPY --from=builder mutatingflow /app/.
EXPOSE 8080
EXPOSE 8443
CMD ["/app/mutatingflow"]
