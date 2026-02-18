FROM debian:stable-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    xvfb x11-utils x11vnc fonts-dejavu-core \
    golang-go ca-certificates build-essential pkg-config \
    xorg-dev mesa-common-dev

WORKDIR /test
COPY . .
RUN go build -o gui .
RUN chmod +x run.sh

EXPOSE 5900

CMD ["/test/run.sh"]
