# Go Add Function Program

这是一个简单的 Go 程序，实现了整数加法功能。

## 功能特性

- `Add(a, b int) int`: 计算两个整数的和
- `AddMultiple(numbers ...int) int`: 计算多个整数的和
- 命令行界面支持
- 完整的单元测试

## 使用方法

### 运行程序

```bash
# 添加两个数字
go run main.go 5 10

# 添加多个数字
go run main.go 1 2 3 4 5
```

### 运行测试

```bash
# 运行所有测试
go test

# 运行测试并显示详细输出
go test -v

# 运行测试并显示覆盖率
go test -cover
```

### 构建程序

```bash
# 构建可执行文件
go build -o add_program

# 运行构建的程序
./add_program 10 20
```

## 示例

```bash
$ go run main.go 5 10
Result: 5 + 10 = 15

$ go run main.go 1 2 3 4 5
Result: [1 2 3 4 5] = 15

$ go run main.go -5 10
Result: -5 + 10 = 5
```

## 测试用例

程序包含全面的测试用例，涵盖：
- 正数相加
- 负数相加
- 正负数混合
- 零值处理
- 大数处理
- 多个数字相加
- 边界情况
