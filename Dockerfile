FROM ubuntu:16.04

WORKDIR /app/

COPY ./build/dp-file-downloader .

CMD ./dp-file-downloader
