FROM ubuntu:14.04
MAINTAINER l00374667 "l00273667@openvmse.org"
ENV REFRESHED_AT 2016-02-17

RUN mkdir -p /opt/service

COPY serv_disc /opt/service/serv_disc

WORKDIR /opt/service

ENTRYPOINT ["./serv_disc"]
