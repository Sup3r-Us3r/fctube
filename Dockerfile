FROM golang:1.23-alpine

RUN apk add --no-cache bash ffmpeg

WORKDIR /app

CMD ["bash"]
