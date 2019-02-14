#build stage
FROM golang:1.11.5-alpine3.9 AS builder
RUN apk add --no-cache git
WORKDIR /go/src/github.com/dgkanatsios/azuregameserversscalingkubernetes
COPY . .
#extra step to copy the HTML stuff
RUN cd ./cmd/apiserver && mkdir -p /build && cp -r html /build/html
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /build/apiserver ./cmd/apiserver

#final stage
FROM alpine:3.9
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY --from=builder /build/apiserver ./
COPY --from=builder /build/html ./html
CMD ["./apiserver"]
EXPOSE 8000 8001
