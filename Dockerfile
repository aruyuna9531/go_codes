# 构建image： docker build -p 9000:9001 -p 9001:9002 -t myserver:1 .

# 指定一个基础镜像，本地没有会从官方库下载（需要指定版本的时候加tag）
# 因为下面用到go build所以下一个golang镜像，如果直接丢可执行程序进去并执行的话，全空容器也可。
# 这里主要是方便从github拉仓库后直接docker build、docker run就能跑起程序（万一拉仓库的人电脑没装go——虽然这概率很低）
FROM golang

# 一个声明（就是写下作者叫啥，没啥卵用，不写也行）
MAINTAINER yuna

WORKDIR /code/golang_server/
ADD ./ ./

# 下面这一行是把时区转为北京时间 docker内日志打印是根据的这个时区
RUN cp /usr/share/zoneinfo/Asia/Shanghai /etc/localtime && echo 'Asia/Shanghai' > /etc/timezone

ENV LANG C.UTF-8

# proxy设置（看情况）
# proxy setting is not necessary. if build failed just fix this line (or add other proxy)
ENV GOPROXY https://goproxy.cn

# 对外暴露的端口（目前不需要 不过反正占个位先）
# docker build的-p参数，冒号后面的值对应这个暴露的端口，前面值对应宿主机端口，其他地方通过访问宿主机的左值端口与这个容器内的进程通信
# -p参数可以指定多个，即暴露多个端口，像第1行那样，此时所有的右值都要在下面定义
EXPOSE 9001 9002

# 这里依赖下载失败可能会导致image创建失败，可以根据情况注掉上面的PROXY多试几次。
# docker image may build failed if import go mod download failed. you can modify PROXY above and try again
RUN go build

# test是go.mod的包名，go build不带参数就生成test可执行文件（懒 反正docker一启你也不用关心它的可执行程序叫阿猫还是阿狗）
ENTRYPOINT ["./test"]

# ENTRYPOINT不要写go run main.go
# go run会生成2个进程，1个就是go run，另一个是go run生成的临时可执行文件，ENTRYPOINT指定的命令是PID=1的进程，因此可执行文件是PID!=1的进程。
# docker容器里只有PID=1的进程会接收docker kill信号，不等于1的进程只能exec进去kill，因此在外面执行docker kill -s="SIGTERM"是无法被可执行程序接收从而优雅退出的，只有容器自己被SIGTERM了。
# 上述写法才会让可执行程序成为PID=1的进程，docker kill sigterm可以优雅关闭容器内程序。