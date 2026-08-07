# landc-go/tools

Go 通用工具集，覆盖日常开发中的常见需求。

## 安装

```bash
go get github.com/LandcLi/landc-go/tools
```

## 包一览

| 包 | 功能 |
|----|------|
| `cache` | 本地内存缓存（LRU + TTL 过期 + 后台清理） |
| `str` | 字符串处理（截取/填充/脱敏/拼音/MD5/Base64/随机/...） |
| `num` | 数字处理（范围限制/格式化/随机/中文大写/...） |
| `generate` | 生成器（UUID / 雪花ID / 安全随机字符串 / 验证码） |
| `converter` | 类型转换（struct↔map / slice / 深拷贝） |
| `email` | 邮箱格式验证与解析 |
| `geo` | IP 地理位置查询（基于 GeoIP2） |
| `mail` | SMTP/IMAP/POP3 邮件收发 |
| `security` | AES-GCM 加解密（`enc:` 前缀 + 明文兼容降级，密钥调用方传入） |
| `ratelimit` | 缓存限流（间隔 / 计数，Cache 接口原子 `Incr` 防并发竞态） |
| `verifycode` | 验证码（生成 + 限流 + TTL 存储 + 一次性校验） |
| `tag` | 结构体 Tag 验证器（required/email/phone/min/max/pattern/enum/...） |

## 使用示例

```go
import (
    "github.com/LandcLi/landc-go/tools/str"
    "github.com/LandcLi/landc-go/tools/generate"
    "github.com/LandcLi/landc-go/tools/cache"
)

// 字符串
str.CamelToSnake("UserName")    // "user_name"
str.MaskPhone("13800138000")    // "138****8000"
str.RandomString(16)            // "aB3xKm9pQwE2..."

// UUID
generate.UUID()                 // "550e8400-e29b-41d4-a716-446655440000"
generate.SnowflakeID()          // 1234567890123456

// 缓存
c := cache.NewGlobalCache()
c.SetWithExpiration("key", "value", 5*time.Minute)
val, ok := c.Get("key")

// AES-GCM 加解密（v0.7.0）
cipher, _ := security.NewCipherFromString("0123456789abcdef0123456789abcdef")
enc, _ := cipher.Encrypt([]byte("sensitive"))
plain, _ := cipher.Decrypt(enc) // 兼容无 enc: 前缀的历史明文

// 缓存限流（v0.7.0，Cache 接口原子自增）
rlCache := ratelimit.NewIntervalLimiter(c)
if !rlCache.Allow("sms:send:"+phone, 60*time.Second) { /* 间隔未到 */ }
cl := ratelimit.NewCountLimiter(c)
if !cl.Allow("login:daily:"+ip, 10, 24*time.Hour) { /* 超每日上限 */ }

// 验证码（v0.7.0）
vc := verifycode.NewManager(c)
code, _ := vc.Generate("sms:13800000000")
if !vc.Verify("sms:13800000000", input) { /* 校验失败 */ }
```
