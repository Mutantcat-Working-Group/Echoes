<div align=center>
<img src="https://s2.loli.net/2024/04/05/xFopiS8CgBswb9y.jpg" style="width:100px;" width="100"/>
<h2>回声</h2>
</div>

### 一、产品概述

- 轻量级的服务器状态播报、状态探针与自动监控告警工具。
- 零依赖运行：单二进制文件，启动后无需任何操作，自动监控服务器状态。
- 支持钉钉机器人、邮件、Server 酱三种通知方式，支持每日定时播报与阈值告警。
- 内置探针模式，可作为被监控端的主动探针，通过 `/info` 接口输出服务器状态。
- 一切信息完全由启动参数控制，重新更改模式需关闭进程后以新参数启动。

核心价值：

- 探针、告警、通知一体，简单部署即可使用。
- `/info` 接口内置状态快照缓存，可承受高并发探针请求。
- 全平台支持，编译产物不依赖 CGO 库，直接构建即可运行。

### 二、功能说明

#### 监控能力

- 存活检查（PING）、内存 / CPU / 硬盘使用率监控、负载率监控、操作系统信息获取。
- 各项指标可配置告警阈值，超阈值自动通知。
- 每日定时播报服务器状态。

#### 通知方式

- `dingbot`：钉钉机器人群通知。
- `mail`：SMTP 邮件通知，按端口自动选择 SSL（465）或明文（25/587）。
- `jiang`：Server 酱 SendKey 通知。

#### 探针模式

- 开启后主动提供服务端口，`/info` 接口返回带缓存的服务器状态快照。
- 单次响应毫秒级，缓存窗口内并发请求直接返回同一份快照。

### 三、安装与下载

从 [Releases](https://github.com/Mutantcat-Working-Group/Echoes/releases) 下载对应平台的二进制可执行程序，直接运行即可。

如没有对应平台的包，可自行用源码编译：本程序不依赖 CGO 库，直接 build 之后运行即可。

如需在 Linux 上创建系统守护进程，进入 `/etc/systemd/system` 新建 `echoes.service`，写入以下内容：

```ini
[Unit]
Description=Echoes service

[Service]
Type=forking
ExecStart=/bin/bash -c "{你的回声程序所在路径} [启动参数] &"
KillMode=process
Restart=on-failure
RestartSec=3s

[Install]
WantedBy=multi-user.target
```

之后执行 `systemctl start echoes` 即可。

### 四、快速上手

#### 启动参数

```text
参数名称                     含义
-help                       是否帮助模式（0/1）
-server_name                服务器名称
-daily_time                 每天自动通知的时间（24小时制）
-interval_time              每次自检的时间间隔（秒）
-loadavg_max_percent        负载率告警阈值最（百分比）
-mem_used_percent           内存告警阈值（百分比）
-cpu_used_percent           CPU使用率告警阈值（百分比）
-pin_enable                 探针模式是否开启（1/0）
-port                       主动服务的端口
-notice_mod                 通知方式（dingbot/mail/jiang）
-token                      通知dingbot的群钉钉机器人token / mail的smtp地址(host:port) / jiang的SendKey
-secret0                    通知dingbot的群钉钉机器人secret / mail的发件密码（或授权码）
-smtp_user                  mail通知的发件账号（仅mail模式需要）
-smtp_from                  mail通知的发件来源邮箱（仅mail模式需要）
-smtp_to                    mail通知的收件邮箱（仅mail模式需要）
-cache_ttl                  /info 接口缓存有效期（秒），0 表示与 interval_time 一致
```

#### 参数默认值

```text
参数名称                     默认值
-help                       0
-server_name                echoes_server_{启动时间}
-daily_time                 09:30
-interval_time              30
-loadavg_max_percent        70
-mem_used_percent           90
-cpu_used_percent           90
-pin_enable                 1
-port                       9966
-notice_mod
-token
-secret0
-smtp_user
-smtp_from
-smtp_to
-cache_ttl                  0
```

#### 注意事项

- 若启动时未开启探针模式也未指定通知模式，程序会直接退出。
- 若启用帮助模式，程序仅查看帮助后直接退出。
- dingbot 模式：`token` 为 access_token，`secret0` 为 secret。
- mail 模式：`token` 为 host:port（如 `smtp.qq.com:465`），`secret0` 为发件密码/授权码，`smtp_user` / `smtp_from` / `smtp_to` 必填。
- jiang 模式：`token` 为 Server 酱 SendKey（https://sct.ftqq.com/ 获取）。

#### 演示参数

```sh
# dingbot 模式
./echoes -server_name echoes_server -daily_time 09:30 -interval_time 30 -loadavg_max_percent 70 -mem_used_percent 90 -cpu_used_percent 90 -pin_enable 1 -port 9966 -notice_mod dingbot -token xxxx -secret0 xxxx
```

```sh
# mail 模式（QQ 邮箱 SSL 示例）
./echoes -server_name echoes_server -daily_time 09:30 -interval_time 30 -loadavg_max_percent 70 -mem_used_percent 90 -cpu_used_percent 90 -pin_enable 1 -port 9966 -notice_mod mail -token smtp.qq.com:465 -secret0 授权码 -smtp_user 123456@qq.com -smtp_from 123456@qq.com -smtp_to admin@example.com
```

```sh
# Server 酱模式
./echoes -server_name echoes_server -daily_time 09:30 -interval_time 30 -loadavg_max_percent 70 -mem_used_percent 90 -cpu_used_percent 90 -pin_enable 1 -port 9966 -notice_mod jiang -token SCTxxxxxx
```

#### 通知形式

![ding_bot.png](./example/ding_bot.png)

### 五、接口说明

#### PING `/ping`

- 说明：检查服务器监控是否存活
- 请求方式：任意
- 返回示例：

```json
{
    "code": 0,
    "msg": "pong"
}
```

#### 服务器信息 `/info`

- 说明：获取当前服务器信息。接口内部对状态采集结果做了内存缓存（默认 TTL = `interval_time` 秒，可用 `cache_ttl` 覆盖），缓存窗口内的并发请求直接返回同一份快照，避免 CPU 采样、磁盘 IO 等重型调用把机器打垮；缓存由后台 goroutine 周期预热，单次响应延迟在毫秒级。
- 请求方式：任意
- 返回示例：

```json
{
    "code": 0,
    "data": {
        "disk": {
            "path": "/",
            "fstype": "xfs",
            "total": 53660876800,
            "free": 42608521216,
            "used": 11052355584,
            "usedPercent": 20.596673485588664,
            "inodesTotal": 26214400,
            "inodesUsed": 385915,
            "inodesFree": 25828485,
            "inodesUsedPercent": 1.4721488952636719
        },
        "system": {
            "MemAll": 3743,
            "MemFree": 2086,
            "MemUsed": 1657,
            "MemUsedPercent": 44.27110833773341,
            "Days": 31,
            "Hours": 13,
            "Minutes": 57,
            "Seconds": 55,
            "CpuUsedPercent": 15.000000020372681,
            "OS": "linux",
            "Arch": "386",
            "CpuCores": 2
        }
    },
    "msg": "success"
}
```

### 六、开发进度

- [X] 检查服务器是否存活（PING）
- [X] 内存使用情况监控/告警
- [X] CPU 使用情况监控/告警
- [X] 硬盘使用情况监控
- [X] 操作系统信息获取
- [X] 负载率过高告警
- [X] 作为探针主动获得服务器状态
- [X] 钉钉机器人群通知
- [X] 邮箱通知
- [X] Server 酱通知
- [X] 缓存信息以支持高并发请求

### 七、历史版本

- https://github.com/tyza66/ServerWatcher-DingBot
- https://github.com/tyza66/ServerWatcher-DingBot-Go
