# feedsystem_video_go

## 项目简介
一款基于Go语言的视频feed流系统。

## 目录结构
1. `cmd/`：程序入口（`main.go`）。
2. `configs/`：YAML 配置文件（`config.yaml`）。  
3. `internal/`：核心逻辑  
   - `account/` 账号模型、仓储、服务  
   - `config/` YAML 配置加载  
   - `db/` MySQL 连接与迁移  
   - `http/` Gin 路由与 handler  
4. `test/`：Postman Collection（`account.json`）  
5. `start.sh`：自定义启动脚本  

## 快速开始
1. 准备 Go 1.24+、MySQL 8+ 环境。  
2. 创建 `feedsystem` 数据库，按需修改 `configs/config.yaml`。  
3. 安装依赖：`go mod tidy`。  
4. 启动服务：`go run ./cmd`（或 `sh start.sh`）。  
5. 导入 `test/account.json`，设置 `host=http://localhost:8080` 可快速调接口。  

## API
| 方法 | 路径 | 说明 |
|------|------|------|
| POST | `/account/register` | `username`,`password` 创建账号 |
| POST | `/account/rename` | `id`,`new_username` 修改账号名 |
| POST | `/account/changePassword` | `id`,`new_password` 修改密码 |
| POST | `/account/findByID` | `id` 查询账号 |
| POST | `/account/findByUsername` | `username` 查询账号 |

- 接口统一使用 JSON 请求/响应。  
- 启动时自动执行 `db.AutoMigrate`，保持用户表结构一致。  
