// Command coverage 合并单元与集成覆盖率，按真实 service 源文件校验语句覆盖率。
package main

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/tools/cover"
)

type blockKey struct {
	file                                 string
	startLine, startCol, endLine, endCol int
}
type count struct{ covered, total int }

func main() {
	root := flag.String("root", ".", "项目根目录")
	out := flag.String("out", ".cache/coverage/merged.out", "合并 profile")
	report := flag.String("report", ".cache/coverage/service.md", "service 报告")
	minimum := flag.Float64("min", 60, "最低整体百分比")
	flag.Parse()
	if err := run(*root, flag.Args(), *out, *report, *minimum); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(root string, profiles []string, out, report string, minimum float64) error {
	if len(profiles) < 2 || minimum < 0 || minimum > 100 {
		return errors.New("必须提供单元与集成 profile 及有效门槛")
	}
	raw, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return err
	}
	module := modfile.ModulePath(raw)
	if module == "" {
		return errors.New("缺少实际模块路径")
	}
	blocks := map[blockKey]cover.ProfileBlock{}
	for _, input := range profiles {
		rows, err := cover.ParseProfiles(input)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return errors.New("空覆盖率文件")
		}
		for _, row := range rows {
			if row.Mode != "atomic" {
				return errors.New("race 覆盖率必须使用 atomic 模式")
			}
			for _, b := range row.Blocks {
				key := blockKey{row.FileName, b.StartLine, b.StartCol, b.EndLine, b.EndCol}
				old, exists := blocks[key]
				if exists && old.NumStmt != b.NumStmt {
					return errors.New("覆盖率块结构不一致，请清理并重新采集")
				}
				b.Count = max(b.Count, old.Count)
				blocks[key] = b
			}
		}
	}
	files := map[string]count{}
	err = filepath.WalkDir(filepath.Join(root, "internal/modules"), func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		name := d.Name()
		if !d.IsDir() && strings.HasPrefix(name, "service") && strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go") {
			rel, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			files[module+"/"+filepath.ToSlash(rel)] = count{}
		}
		return nil
	})
	if err != nil {
		return err
	}
	keys := slices.Collect(maps.Keys(blocks))
	slices.SortFunc(keys, func(a, b blockKey) int {
		if n := strings.Compare(a.file, b.file); n != 0 {
			return n
		}
		for _, pair := range [][2]int{{a.startLine, b.startLine}, {a.startCol, b.startCol}, {a.endLine, b.endLine}, {a.endCol, b.endCol}} {
			if pair[0] < pair[1] {
				return -1
			}
			if pair[0] > pair[1] {
				return 1
			}
		}
		return 0
	})
	var merged strings.Builder
	merged.WriteString("mode: atomic\n")
	for _, k := range keys {
		b := blocks[k]
		fmt.Fprintf(&merged, "%s:%d.%d,%d.%d %d %d\n", k.file, k.startLine, k.startCol, k.endLine, k.endCol, b.NumStmt, b.Count)
		if c, ok := files[k.file]; ok {
			c.total += b.NumStmt
			if b.Count > 0 {
				c.covered += b.NumStmt
			}
			files[k.file] = c
		}
	}
	var result strings.Builder
	result.WriteString("# service 语句覆盖率\n\n单元与隔离集成按语句块取并集；仅 internal/modules 下 service*.go，排除测试及生成物。门槛按整体统计。\n\n| 文件 | 覆盖/语句 | 百分比 |\n|---|---|---|\n")
	var total count
	for _, file := range slices.Sorted(maps.Keys(files)) {
		c := files[file]
		if c.total == 0 {
			return fmt.Errorf("service 源文件未采集：%s", file)
		}
		fmt.Fprintf(&result, "| %s | %d/%d | %.2f%% |\n", strings.TrimPrefix(file, module+"/"), c.covered, c.total, 100*float64(c.covered)/float64(c.total))
		total.covered += c.covered
		total.total += c.total
	}
	if total.total == 0 {
		return errors.New("没有 service 可执行语句")
	}
	percent := 100 * float64(total.covered) / float64(total.total)
	fmt.Fprintf(&result, "\n整体：%d/%d = %.2f%%；门槛：%.2f%%。\n", total.covered, total.total, percent, minimum)
	for path, data := range map[string]string{out: merged.String(), report: result.String()} {
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(data), 0600); err != nil {
			return err
		}
	}
	fmt.Print(result.String())
	if percent < minimum {
		return fmt.Errorf("service 覆盖率 %.2f%% 未达到 %.2f%%", percent, minimum)
	}
	return nil
}
