FROM alpine:3.8
RUN apk --no-cache add ca-certificates
WORKDIR /app
COPY /bin/apiserver ./
COPY /cmd/apiserver/html ./html
CMD ["./apiserver"]
EXPOSE 8000 8001
