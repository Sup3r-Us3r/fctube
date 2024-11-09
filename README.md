# FCTube

## Overview

This Go application has two parts. The first part is responsible for splitting an mp4 video into several chunks, which are then sent to an external Docker volume at `/media/uploads`.

The second part of the application converts these chunks into a final file, which is then converted to mp4-dash format to be used on the web.

- [x] Golang
- [x] RabbitMQ
- [x] Nginx
- [x] PostgreSQL
- [x] FFmpeg
- [x] Docker + Docker Compose

---

## Setup

Access the folder:

```sh
$ cd fctube
```

Create Docker volume:

```sh
$ docker volume create external-storage
```

Up Container:

```sh
$ docker-compose up -d
```

Access container:

```sh
$ docker exec -it go_app_dev bash
```

Install dependencies:

```sh
$ go mod tidy
```

## Run Split Chunks

The first part of the application splits the video into several 1MB chunks,
which are then sent to `/media/uploads`.

Run:

```sh
# usage: go run cmd/split_chunks/main.go <mp4-file-path> <output-folder-name>
$ go run cmd/split_chunks/main.go mediatest/media/uploads/1/video_example.mp4 1
```

> `<mp4-file-path>` is the path to the video you want to split into chunks.

> `<output-folder-name>` is the name of the folder where these chunks will be stored.
> In the example, I used "1" as the name, and if I use another video, I will simply increment the name by +1.

## Run Video Converter

The second part of the application consists of converting those chunks into a
final video and then converting it to `mp4-dash`.

Run:

```sh
$ go run cmd/video_converter/main.go
```

At this point, the application will wait for a message to be published in the `video_conversion_queue`
queue with the routing key `conversion` through the `conversion_exchange` exchange.

Then, manually send a message using RabbitMQ by accessing http://localhost:15672
with the username `guest` and password `guest`.

Now, go to the `Exchanges` tab and publish a message with the routing key `conversion`:

```json
{ "video_id": 1, "path": "/media/uploads/1" }
```

> `video_id`: The unique identifier for the video.

> `path`: The path where the video chunks were stored. **/media/uploads** will
> always be the base path, and **1** is the name of the folder you assigned to
> store the chunks in the previous step.

## Preview

To preview the playback of the converted `mp4-dash` video, there is a demo
available in the [mediatest/html/player.html](mediatest/html/player.html) file.
Just edit the `url` variable on line **28**, replacing the number **1** with the folder
name you used to store the chunks.

Now, open this `.html` file in your browser to view the playback.
The content is being delivered via Nginx, which is reading and serving these files.
