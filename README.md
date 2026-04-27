# 松壳应急值班系统

一个现代化的值班排班管理系统，支持多组多平台的值班人员管理和大屏展示。

## 功能特性

- 📅 **日历排班**：直观的日历界面，支持点击日期进行排班
- 👥 **人员管理**：按组管理值班人员信息
- 🖥️ **大屏展示**：实时显示今日值班人员，支持表格式布局
- 📱 **响应式设计**：支持桌面和移动设备访问
- 🎨 **现代化UI**：采用玻璃态设计，支持深色主题

## 技术栈

### 后端
- **语言**：Go 1.21+
- **框架**：Gin
- **数据库**：SQLite (GORM)
- **API**：RESTful

### 前端
- **框架**：Vue 3 + TypeScript
- **构建工具**：Vite
- **UI组件**：Element Plus
- **路由**：Vue Router
- **HTTP客户端**：Axios
- **日期处理**：Day.js

## 项目结构

```
oncallSystem/
├── server/              # 后端代码
│   ├── api/            # API处理器
│   ├── model/          # 数据模型
│   ├── main.go         # 主程序入口
│   └── oncall.db       # SQLite数据库（运行时生成）
├── web-ui/             # 前端代码
│   ├── src/
│   │   ├── views/      # 页面组件
│   │   ├── router/     # 路由配置
│   │   └── main.ts     # 入口文件
│   ├── dist/           # 打包输出目录
│   └── package.json
└── README.md
```

## 快速开始

### 前置要求

- Go 1.21 或更高版本
- Node.js 18 或更高版本
- npm 或 yarn

### 安装步骤

#### 1. 克隆项目

```bash
git clone <repository-url>
cd oncallSystem
```

#### 2. 启动后端

```bash
cd server
go run main.go
```

后端服务将在 `http://localhost:8888` 启动

#### 3. 启动前端（开发模式）

```bash
cd web-ui
npm install
npm run dev
```

前端开发服务器将在 `http://localhost:5173` 启动

#### 4. 访问应用

- **管理后台**：http://localhost:5173
- **大屏展示**：http://localhost:5173/dashboard

## 数据结构

### 组（Groups）
系统包含6个组：
- 运维
- 大厅
- 新娱乐
- 本地
- 公共
- 国际

### 平台（Platforms）
系统包含12个平台：

**运维组专用（主/备）：**
- Idn-主、Idn-备
- Idn-Sub-主、Idn-Sub-备
- Malaysia-主、Malaysia-备

**其他组专用（服务端/客户端）：**
- Idn-服务端、Idn-客户端
- Idn-Sub-服务端、Idn-Sub-客户端
- Malaysia-服务端、Malaysia-客户端

## 使用说明

### 管理后台

1. **查看日历**：首页显示当月日历，已排班的日期会显示人数
2. **编辑排班**：点击任意日期打开排班对话框
3. **选择人员**：为每个组的每个平台选择值班人员（单选）
4. **保存排班**：点击保存按钮提交排班信息
5. **人员管理**：点击"人员管理"按钮批量管理各组人员

### 大屏展示

1. **实时显示**：自动显示今日值班人员
2. **历史查看**：点击右侧日历可查看历史排班
3. **表格布局**：
   - 运维组：显示"主"和"备"两列
   - 其他组：显示"服务端"和"客户端"两列
4. **返回实时**：点击"返回今日实时"按钮回到今日视图

## 生产部署

### 打包前端

```bash
cd web-ui
npm run build
```

打包后的文件位于 `web-ui/dist` 目录

### 部署方案

详细的部署说明请参考 [部署说明.md](./部署说明.md)，包括：
- Nginx 配置
- 后端服务配置
- HTTPS 配置
- 性能优化建议

## API 接口

### 组管理
- `GET /api/groups` - 获取所有组

### 平台管理
- `GET /api/platforms` - 获取所有平台

### 人员管理
- `GET /api/people` - 获取所有人员
- `POST /api/people/batch` - 批量更新人员

### 排班管理
- `GET /api/shifts?start=YYYY-MM-DD&end=YYYY-MM-DD` - 获取排班
- `POST /api/shifts` - 创建/更新排班

### 大屏数据
- `GET /api/dashboard/today` - 获取今日值班数据

## 开发说明

### 后端开发

```bash
cd server
go run main.go
```

数据库会自动初始化并填充测试数据。

### 前端开发

```bash
cd web-ui
npm run dev
```

修改代码后会自动热重载。

### 数据库重置

如需重置数据库：

```bash
cd server
rm oncall.db
go run main.go
```

## 更新日志

### v2.0.0 (2026-02-10)
- ✨ UI优化：表格式布局
- ✨ 平台扩展：从6个扩展到12个
- ✨ 子类型支持：运维组使用主/备，其他组使用服务端/客户端
- 🔧 所有组改为单选
- 🎨 改进大屏展示样式

### v1.0.0 (2026-02-09)
- 🎉 初始版本发布
- ✨ 基础排班功能
- ✨ 人员管理
- ✨ 大屏展示

## 许可证

MIT License

## 联系方式

如有问题或建议，请联系开发团队。
