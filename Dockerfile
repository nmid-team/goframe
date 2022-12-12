FROM ubuntu

WORKDIR /root

ARG TARGETARCH

COPY server ./server
#COPY config.yaml .
#COPY db ./db
#COPY library ./library

#RUN set -xe \
  ## disable sshd
  ## && rm -r /etc/service/sshd

# 镜像启动服务自动被拉起配置
COPY run /etc/service/run
RUN chmod +x /etc/service/run

# dockerfile 中不允许定义 CMD。镜像启动需要执行基础定义逻辑