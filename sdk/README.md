# 内嵌接口

本 fork 基于上游 v0.4.2。`sdk/render` 复用现有加载、变量解析和渲染；`sdk/format` 提供无文件写入的 HCL/模板格式化。

```go
result, err := render.Render(render.Options{
    Path: "packs/traefik",
    VariableFiles: []string{"private.hcl"},
    Format: true,
})
```

变量来源必须显式传入，SDK 不自动读取环境变量。诊断经 `Result.Diagnostics` 返回，调用者将 warning 与渲染内容分开输出。
BaseDir 可指定 fileContents 的相对文件基准；默认保持当前目录语义，SDK 不修改进程 cwd。
未声明的输入被忽略并 warning；语法、已声明类型和必填错误仍失败。上游变量优先级和模板缺失字段语义保持不变。

上游接点保持独立：

- `Variable.Merge` 的声明约束检查是单独的缺陷修复提交。
- 未声明变量诊断降为 warning，Manager 暴露已有诊断，CLI 展示 warning。
- 文件中未知依赖路径进入已有 merge 流程，统一遵循 `IgnoreMissingVars`。
- 原 fmt 命令委托给 format SDK，模板扫描识别字符串、注释和多行片段。

源码获取、缓存、部署策略、终端展示与资产分发属于消费方。跟进上游时优先删除已被上游覆盖的小补丁，不复制或重组上游内部包。
