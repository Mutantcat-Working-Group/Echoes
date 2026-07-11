### 1.0.20260711
  - 修复了`-help` flag 定义在`flag.Parse()`之后导致帮助模式永远不生效的问题
  - 修复了`RegisterRouter`里循环中`InitRouter`返回值被覆盖、多个路由错误处理不正确的问题
  - 将启动通知文本`SendTextMessage`,`SendMail`,`Jiang.Send`改为异步发送，避免启动阶段被网络调用卡住
  - 补充 README 中未实现的邮箱通知（`mail`模式）和 Server 酱通知（`jiang`模式）
  - `mail` 模式基于标准库 `net/smtp` + `crypto/tls`，支持 SSL（465）与明文（25/587），无需新增外部依赖
  - `jiang` 模式基于标准库 `net/http`，对接 https://sct.ftqq.com/ 统一推送接口
  - 新增启动参数：`-smtp_user`、`-smtp_from`、`-smtp_to`（mail 模式专用）
  - `token` / `secret0` 语义扩展为三模式复用：dingbot（token/secret0）、mail（host:port / 密码）、jiang（SendKey）
  - 更新 README 参数表、默认值、注意事项及三种模式演示命令，并勾选开发进度

### 1.0.20240405
  - 实现了基本的功能
