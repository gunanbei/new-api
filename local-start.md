```bash
# 启动后端
# 设置环境变量
source ./dev-env.sh
go run main.go
# 访问http://localhost:3000


#启动前端
cd web
bun install --filter ./default
cd default
bun run dev -- --host 0.0.0.0 --port 5173
#访问 http://localhost:5173，前端请求会自动代理到 http://localhost:3000。

#注意：
#前端修改会自动热更新。
#Go 后端修改后需要停止并重新执行 go run main.go。
