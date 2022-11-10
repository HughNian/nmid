FROM docker

WORKDIR /root

ARG TARGETARCH

COPY cmd/server/config ./config
COPY cmd/server/nmid ./nmid

# 镜像启动服务自动被拉起配置
COPY run /etc/service/run
RUN chmod +x /etc/service/run