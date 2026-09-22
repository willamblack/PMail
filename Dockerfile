FROM node:22-alpine as febuild
WORKDIR /work

COPY fe .

RUN yarn install --frozen-lockfile && yarn build


FROM golang:1.26.8-alpine@sha256:8ac98ca534ac3f51e1f420a1dd2c15e74c75cfa0f23f3ad27eb5d7236c349a0c as serverbuild
ARG VERSION
WORKDIR /work
COPY . .
COPY --from=febuild /work/dist /work/server/listen/http_server/dist
RUN apk update && apk add git
RUN cd /work/server && go build -ldflags "-s -w -X 'main.version=${VERSION}' -X 'main.goVersion=$(go version)' -X 'main.gitHash=$(git show -s --format=%H)' -X 'main.buildTime=$(TZ=UTC-8 date +%Y-%m-%d" "%H:%M:%S)'" -o pmail main.go
RUN cd /work/server/hooks/wechat_push && go build -ldflags "-s -w" -o output/wechat_push wechat_push.go
RUN cd /work/server/hooks/spam_block && go build -ldflags "-s -w" -o output/spam_block spam_block.go


FROM alpine

# PMail derives its runtime root from the executable directory. The binary is
# installed as /work/pmail, so persistent runtime data belongs in /work/config.
# This is different from the source-tree path server/config used at build time.
WORKDIR /work

# 设置时区
RUN apk add --no-cache tzdata ca-certificates \
    && ln -sf /usr/share/zoneinfo/Asia/Shanghai /etc/localtime \
    && echo "Asia/Shanghai" > /etc/timezone


COPY --from=serverbuild /work/server/pmail .
COPY --from=serverbuild /work/server/hooks/wechat_push/output/* ./plugins/
COPY --from=serverbuild /work/server/hooks/spam_block/output/* ./plugins/

EXPOSE 25 80 110 443 465 587 995 993

CMD ["/work/pmail"]
