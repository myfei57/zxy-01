# EdgeTranscode

EdgeTranscode 是短视频平台的内容处理服务，接收创作者上传的原始视频，转码为
720p/1080p 多码率分片，质检通过后构建播放清单并发布到 CDN，播放器按清单拉流，
运营控制台提供任务、流、节点和质量页面。

## 构建与运行

```bash
go build -mod=vendor ./...
go test -mod=vendor -count=1 ./...
go vet -mod=vendor ./...
```

启动服务：

```bash
go run ./cmd/transcode -addr :8080 -data ./data
```

健康检查：`curl http://localhost:8080/healthz`

## Docker

```bash
bash build_benzhi_docker.sh edge-transcode linux/amd64
docker run --rm -p 8080:8080 edge-transcode bash -c 'go run ./cmd/transcode -addr :8080 -data /tmp/data'
```
