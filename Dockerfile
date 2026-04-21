FROM golang:1.25.1-bullseye AS build

WORKDIR /go/src/sei-load

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /go/bin/seiload ./

FROM gcr.io/distroless/base
COPY --from=build /go/bin/seiload /usr/bin/

ENTRYPOINT ["/usr/bin/seiload"]