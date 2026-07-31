FROM hub-dev.hexin.cn/baseimages/alpine

ARG BRANCH
ARG REVISION
ARG CURRENT_TIME

LABEL org.opencontainers.image.ref.name="${BRANCH}" \
      org.opencontainers.image.revision="${REVISION}" \
      org.opencontainers.image.created="${CURRENT_TIME}"

RUN apk add --no-cache ca-certificates tzdata \
    && mkdir -p /app/etc

ENV TZ=Asia/Shanghai

WORKDIR /app

COPY output/build/app /app/app

EXPOSE 43999
STOPSIGNAL SIGTERM

ENTRYPOINT ["/app/app"]
CMD ["-f", "/app/cubesandboximageserver-api.yaml"]
