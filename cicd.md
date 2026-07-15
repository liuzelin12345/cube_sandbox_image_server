# CI 流水线流程
1、golang构建配置
name: {{ .name }}
image: 'hub-dev.hexin.cn/pack-ci-plugin/baseimages/golang:{{ .version }}'
environment:
  GOPROXY: http://repositories-public.myhexin.com:8087/repository/go-public,direct
  GO111MODULE: "on"
  CGO_ENABLED: 0
  GOOS: "linux"
  GOARCH: "amd64"
  GOSUMDB: "off"
  GOINSECURE: gitlab.myhexin.com
  GOPRIVATE: gitlab.myhexin.com
commands:
  - mkdir -p ./output/build/
  - go build -o ./output/build/app

2、容器镜像配置
name: {{ .name }}
image: hub-dev.hexin.cn/pack-ci-plugin/docker-build
settings:
  dockerfile_path: {{ .dockerfile_path }}
  image_group: {{ .image_group }}
  image_name: {{ .image_name }}
  tag: {{ .tag }}
  pull_always: {{ .pull_always }}
  no_cache: {{ .no_cache }}
  online_dockerfile: {{ .online_dockerfile }}
  docker_context: {{ .context }}
  build_args_from_env: 
  - BRANCH
  - REVISION
  - CURRENT_TIME
volumes:
- name: docker
  path: /var/run/docker.sock