# Redis Streams Manager

一个用于管理Redis Streams的命令行工具，支持查看、添加和删除流及其消息。

## 功能特性

- **查看流**：列出所有流或特定流的内容
- **添加消息**：直接添加消息或从文件读取内容添加
- **删除操作**：删除整个流或特定消息
- **环境配置**：通过环境变量配置Redis连接参数
- **友好输出**：格式化输出，带有emoji反馈

## 安装

### 前提条件

- Go 1.16+
- Redis服务器

### 编译

```bash
cd tools/streams-manager
go build -o streams-manager
```

在Windows上：

```powershell
cd tools/streams-manager
go build -o streams-manager.exe
```

## 使用方法

### 基本命令

```bash
# 查看所有流
streams-manager ls

# 查看特定流的内容
streams-manager ls <StreamName>

# 添加消息到流
streams-manager add <StreamName> '{"key": "value", "name": "test"}'

# 从文件读取内容添加到流
streams-manager add <StreamName> -f message.json

# 删除整个流
streams-manager del <StreamName>

# 删除特定消息
streams-manager del <StreamName> <MessageID>

# 显示帮助信息
streams-manager help
```

### 环境变量

可以通过环境变量配置Redis连接：

```bash
# 设置Redis主机（默认：localhost）
export REDIS_HOST=redis.example.com

# 设置Redis端口（默认：6379）
export REDIS_PORT=6379

# 设置Redis密码（可选）
export REDIS_PASSWORD=your_password
```

在Windows PowerShell中：

```powershell
$env:REDIS_HOST="redis.example.com"
$env:REDIS_PORT="6379"
$env:REDIS_PASSWORD="your_password"
```

## 示例

### 查看所有流

```bash
streams-manager ls
```

输出示例：
```
=== Redis Streams ===

Stream: mystream
----------------------------------------
Length: 5
Last ID: 1774756676541-0
Consumer Groups (1):
  - mygroup: 0 pending, 2 consumers
```

### 查看特定流的内容

```bash
streams-manager ls mystream
```

输出示例：
```
Messages in stream 'mystream' (showing last 100):
================================================================================

Message 1:
--------------------------------------------------------------------------------
ID: 1774756676541-0
  key                 : value
  name                : test

Message 2:
--------------------------------------------------------------------------------
ID: 1774756676540-0
  key                 : another_value
  name                : another_test
================================================================================
Total messages: 2
```

### 添加消息

```bash
streams-manager add mystream '{"key": "value", "name": "test", "timestamp": "2024-01-01"}'
```

输出示例：
```
✅ Message added successfully with ID: 1774756676542-0
```

### 从文件添加消息

创建 `message.json` 文件：
```json
{
  "key": "file_value",
  "name": "file_test",
  "source": "file"
}
```

然后运行：
```bash
streams-manager add mystream -f message.json
```

输出示例：
```
✅ Message from file 'message.json' added successfully with ID: 1774756676543-0
```

### 删除消息

```bash
streams-manager del mystream 1774756676541-0
```

输出示例：
```
✅ Message with ID '1774756676541-0' deleted successfully
```

### 删除整个流

```bash
streams-manager del mystream
```

输出示例：
```
✅ Stream 'mystream' deleted successfully
```

## 技术栈

- **语言**：Go
- **Redis客户端**：go-redis/redis/v8
- **功能**：Redis Streams操作

## 注意事项

1. 确保Redis服务器正在运行且可访问
2. 消息内容必须是有效的JSON格式
3. 从文件读取时，文件内容也必须是有效的JSON格式
4. 删除操作不可逆，请谨慎使用

## 许可证

MIT License
