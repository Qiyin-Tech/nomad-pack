// Copyright IBM Corp. 2021, 2025
// SPDX-License-Identifier: MPL-2.0

// Package format formats HCL and unrendered Pack templates without executing them.
package format

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/hcl/v2/hclwrite"
)

// Source formats .hcl or .tpl source. Template actions are preserved verbatim;
// only their surrounding HCL layout is formatted. No files are read or written.
func Source(name string, src []byte) ([]byte, error) {
	switch filepath.Ext(name) {
	case ".hcl":
		_, diags := hclsyntax.ParseConfig(src, name, hcl.InitialPos)
		if diags.HasErrors() {
			return nil, diags
		}
		return hclwrite.Format(src), nil
	case ".tpl":
		return template(string(src))
	default:
		return nil, fmt.Errorf("不支持格式化 %s：需要 .hcl 或 .tpl 文件", name)
	}
}

func template(src string) ([]byte, error) {
	prefix := "__NOMAD_PACK_TPL_"
	for strings.Contains(src, prefix) {
		prefix += "_"
	}
	var masked strings.Builder
	var replacements []string
	for {
		start := strings.Index(src, "[[")
		if start < 0 {
			masked.WriteString(src)
			break
		}
		end, err := actionEnd(src, start+2)
		if err != nil {
			return nil, err
		}
		placeholder := prefix + strconv.Itoa(len(replacements)/2) + "__"
		masked.WriteString(src[:start])
		masked.WriteString(placeholder)
		replacements = append(replacements, placeholder, src[start:end])
		src = src[end:]
	}
	formatted := hclwrite.Format([]byte(masked.String()))
	return []byte(strings.NewReplacer(replacements...).Replace(string(formatted))), nil
}

// actionEnd recognizes Go template strings and comments so embedded ]] does not
// end an action. QuotedPrefix uses Go's own string and rune literal rules.
func actionEnd(src string, start int) (int, error) {
	for i := start; i < len(src); {
		switch {
		case strings.HasPrefix(src[i:], "]]"):
			return i + 2, nil
		case strings.HasPrefix(src[i:], "/*"):
			end := strings.Index(src[i+2:], "*/")
			if end < 0 {
				return 0, fmt.Errorf("模板注释未闭合")
			}
			i += end + 4
		case strings.ContainsRune("\"'`", rune(src[i])):
			quoted, err := strconv.QuotedPrefix(src[i:])
			if err != nil {
				return 0, fmt.Errorf("模板字符串无效：%w", err)
			}
			i += len(quoted)
		default:
			i++
		}
	}
	return 0, fmt.Errorf("模板片段未闭合：缺少 ]]")
}
