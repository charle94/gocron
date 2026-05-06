# gocron - 定时任务管理系统
[![Downloads](https://img.shields.io/github/downloads/ouqiang/gocron/total.svg)](https://github.com/ouqiang/gocron/releases)
[![license](https://img.shields.io/github/license/mashape/apistatus.svg?maxAge=2592000)](https://github.com/ouqiang/gocron/blob/master/LICENSE)
[![Release](https://img.shields.io/github/release/ouqiang/gocron.svg?label=Release)](https://github.com/ouqiang/gocron/releases)

# 项目简介
使用Go语言开发的轻量级定时任务集中调度和管理系统, 用于替代Linux-crontab [查看文档](https://github.com/ouqiang/gocron/wiki)

原有的延时任务拆分为独立项目[延迟队列](https://github.com/ouqiang/delay-queue)

## 功能特性
* Web界面管理定时任务
* crontab时间表达式, 精确到秒
* 任务执行失败可重试
* 任务执行超时, 强制结束
* 任务依赖配置, A任务完成后再执行B任务
* 账户权限控制
* 任务类型
    * shell任务：在任务节点上执行shell命令, 支持任务同时在多个节点上运行
    * HTTP任务：访问指定的URL地址, 由调度器直接执行, 不依赖任务节点
* 查看任务执行结果日志
* 任务执行结果通知, 支持邮件、Slack、Webhook
* **SQLite 支持**：无需外部数据库，可直接以单二进制文件运行（纯 Go 实现，无 CGO 依赖）
* **Casbin 多租户 RBAC**：基于域的细粒度权限控制
* **AI 智能助手**：集成 LLM 对话交互，支持自定义 OpenAI 兼容端点，工具调用查看/执行任务（含高危操作确认）
* **任务搜索增强**：支持精确/前缀/后缀/包含/正则多种匹配方式

### 支持平台
> Windows、Linux、Mac OS（包括 Apple Silicon ARM64）

### 环境要求
> MySQL / PostgreSQL / **SQLite**（SQLite 模式下无需外部数据库）

## 下载
[releases](https://github.com/ouqiang/gocron/releases)

## 安装

### 二进制安装
1. 解压压缩包
2. `cd 解压目录`
3. 启动
* 调度器启动
  * Windows: `gocron.exe web`
  * Linux、Mac OS:  `./gocron web`
* 任务节点启动, 默认监听0.0.0.0:5921
  * Windows:  `gocron-node.exe`
  * Linux、Mac OS:  `./gocron-node`
4. 浏览器访问 http://localhost:5920

### 源码编译

```bash
# 1. 安装 Go 1.12+
# 2. 克隆仓库
git clone https://github.com/charle94/gocron
cd gocron

# 3. 当前平台静态编译（推荐，无需 CGO）
make build-static
# 输出: bin/gocron  bin/gocron-node

# 4. 启动
./bin/gocron web
./bin/gocron-node
```

### 跨平台编译

本项目使用纯 Go SQLite（`glebarez/go-sqlite`），**无需 CGO**，支持零配置跨平台交叉编译。

```bash
# macOS Apple Silicon (ARM64)
make build-mac-arm
# 输出: bin/gocron-darwin-arm64  bin/gocron-node-darwin-arm64

# Linux AMD64
make build-linux-amd64
# 输出: bin/gocron-linux-amd64  bin/gocron-node-linux-amd64

# 全部架构一键编译
make build-cross
```

> 以上命令可在任意平台执行，无需目标平台环境。

### docker

```shell
docker run --name gocron --link mysql:db -p 5920:5920 -d ouqg/gocron
```

配置: /app/conf/app.ini  
日志: /app/log/cron.log

### 开发

```bash
# 安装前端依赖
make install-vue

# 同时启动 gocron 和 gocron-node
make run

# 启动前端开发服务器（访问 http://localhost:8080）
make run-vue
```

## 命令行参数

* `gocron -v` 查看版本
* `gocron web`
    * `--host` 默认0.0.0.0
    * `-p` 端口, 默认5920
    * `-e` 运行环境 dev|test|prod，默认prod
* `gocron-node`
    * `-allow-root` *nix平台允许以root用户运行
    * `-s ip:port` 监听地址
    * `-enable-tls` 开启TLS
    * `-ca-file / -cert-file / -key-file` TLS证书

## AI 智能助手

AI 助手集成 OpenAI 兼容的 LLM 接口，支持通过自然语言管理定时任务。

### 配置步骤
1. 以管理员登录，进入「AI助手」→「AI配置」
2. 填写配置：
   - **API 端点**：如 `https://api.openai.com/v1`，也支持 Ollama（`http://localhost:11434/v1`）、Azure OpenAI 等
   - **API Key**：对应平台密钥
   - **模型**：如 `gpt-4o`、`qwen-plus`、`deepseek-chat` 等
   - **高危操作模式**：`确认模式`（默认，执行前需用户确认）或 `自动模式`
3. 开启并保存

### 可操作内容

| 操作 | 是否高危 |
|------|---------|
| 查看任务列表/详情 | 否 |
| 查看节点列表 | 否 |
| 查看任务执行日志 | 否 |
| **立即执行任务** | **是** |
| 启用任务 | 否 |
| **停用任务** | **是** |

### 示例
```
用户: 显示所有启用的任务
用户: 查看任务 ID=5 的最近10条日志
用户: 帮我执行"数据库备份"任务
用户: 列出所有节点
```

## Casbin 多租户权限管理

本项目集成 [Casbin](https://casbin.org/) 基于域的 RBAC 权限模型。

* 配置文件：`conf/rbac_model.conf`（模型定义）、`conf/policy.csv`（默认策略）
* 内置角色：`admin`（完整权限）、`operator`（读写任务/日志）、`viewer`（只读）
* 管理 API（仅管理员）：`GET /admin/policy`、`POST /admin/policy/add`、`POST /admin/role/add` 等

## To Do List
- [x] 版本升级
- [x] 批量开启、关闭、删除任务
- [x] 调度器与任务节点通信支持https
- [x] 任务分组
- [x] 多用户
- [x] 权限控制
- [x] SQLite 支持（无 CGO 静态编译）
- [x] Casbin 多租户 RBAC
- [x] AI 智能助手（LLM 对话）
- [x] 任务搜索多种匹配方式
- [x] 跨平台静态编译（macOS ARM64 / Linux AMD64）

## 使用的组件
* Web框架 [Macaron](http://go-macaron.com/)
* 定时任务调度 [Cron](https://github.com/robfig/cron)
* ORM [Xorm](https://github.com/go-xorm/xorm)
* UI框架 [Element UI](https://github.com/ElemeFE/element)
* RPC框架 [gRPC](https://github.com/grpc/grpc)
* 权限管理 [Casbin](https://casbin.org/)
* SQLite [glebarez/go-sqlite](https://github.com/glebarez/go-sqlite)（纯 Go，无 CGO）

## 反馈
提交[issue](https://github.com/ouqiang/gocron/issues/new)

## ChangeLog

v1.6
--------
* 支持 SQLite（纯 Go，无 CGO，可静态编译）
* 集成 Casbin 多租户 RBAC 权限管理
* 新增 AI 智能助手（OpenAI 兼容端点 + 工具调用 + 高危操作确认）
* 任务搜索支持多种匹配方式
* 跨平台静态编译（macOS ARM64 / Linux AMD64）

v1.5
--------
* 前端使用Vue+ElementUI重构
* 任务通知：WebHook通知、自定义通知模板、关键字匹配通知
* 任务列表页显示下次执行时间

v1.4
--------
* HTTP任务支持POST请求
* 后台手动停止运行中的shell任务
* 任务执行失败重试间隔时间支持用户自定义
* 修复API接口调用报403错误

v1.3
--------
* 支持多用户登录
* 增加用户权限控制

v1.2.2
--------
* 用户登录页增加图形验证码
* 支持从旧版本升级
* 任务批量开启、关闭、删除
* 调度器与任务节点支持HTTPS双向认证
* 修复任务列表页总记录数显示错误

v1.1
--------
* 任务可同时在多个节点上运行
* *nix平台默认禁止以root用户运行任务节点
* 子任务命令中增加预定义占位符, 子任务可根据主任务运行结果执行相应操作
* 删除守护进程模块
* Web访问日志输出到终端
