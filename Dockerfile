FROM golang:1.22-bullseye as build
WORKDIR /app
COPY . .
RUN apt-get update && apt-get install -y ffmpeg
RUN go build -o app .

FROM debian:bullseye-slim
WORKDIR /app
COPY --from=build /app/app /app/app
COPY --from=build /usr/bin/ffmpeg /usr/bin/ffmpeg
EXPOSE 8080
CMD ["./app"]
