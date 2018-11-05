FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY /bin/controller ./
CMD ["./controller"]