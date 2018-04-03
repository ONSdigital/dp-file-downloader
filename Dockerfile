FROM onsdigital/dp-concourse-tools-ubuntu

WORKDIR /app/

COPY ./build/dp-file-downloader .

CMD ./dp-file-downloader
