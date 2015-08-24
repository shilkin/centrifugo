FROM centos:7

ENV VERSION 0.2.3

ENV DOWNLOAD https://github.com/shilkin/centrifugo/releases/download/v$VERSION/centrifugo-$VERSION-linux-amd64.zip

RUN curl -sSL "$DOWNLOAD" -o /tmp/centrifugo.zip && \
    yum install -y unzip && \
    unzip -jo /tmp/centrifugo.zip -d /tmp/ && \
    mv /tmp/centrifugo /usr/bin/centrifugo && \
    rm -f /tmp/centrifugo.zip && \
    echo "centrifugo - nofile 65536" >> /etc/security/limits.d/centrifugo.nofiles.conf

RUN groupadd -r centrifugo && useradd -r -g centrifugo centrifugo

RUN mkdir /centrifugo && chown centrifugo:centrifugo /centrifugo && \
    mkdir /var/log/centrifugo && chown centrifugo:centrifugo /var/log/centrifugo && \
    mkdir -p /opt/centrifugo/web

ADD extras/web/app /opt/centrifugo/web

VOLUME ["/centrifugo", "/var/log/centrifugo"]

WORKDIR /centrifugo

USER centrifugo

CMD ["centrifugo"]

EXPOSE 8000