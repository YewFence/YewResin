package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"filippo.io/age"
	"github.com/joho/godotenv"
	"golang.org/x/term"
)

// ConfigFileEntry 描述一个待导出/导入的配置文件
type ConfigFileEntry struct {
	ArchiveName  string `json:"archive_name"`  // 归档内文件名
	OriginalPath string `json:"original_path"` // 原始绝对路径
	Source       string `json:"source"`        // 路径来源说明
	Description  string `json:"description"`   // 文件描述
}

// Manifest 归档元数据，作为 tar 的第一个条目存储
type Manifest struct {
	Version         int               `json:"version"`
	CreatedAt       string            `json:"created_at"`
	YewResinVersion string            `json:"yewresin_version"`
	Files           []ConfigFileEntry `json:"files"`
}

// =================== 路径探测 ===================

func defaultRcloneConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		appData := os.Getenv("APPDATA")
		if appData == "" {
			return "", fmt.Errorf("APPDATA 未设置")
		}
		return filepath.Join(appData, "rclone", "rclone.conf"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".config", "rclone", "rclone.conf"), nil
}

func defaultKopiaConfigPath() (string, error) {
	if runtime.GOOS == "windows" {
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA 未设置")
		}
		return filepath.Join(localAppData, "kopia", "repository.config"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("获取用户主目录失败: %w", err)
	}
	return filepath.Join(home, ".config", "kopia", "repository.config"), nil
}

// detectConfigFiles 探测所有可导出的配置文件
// 返回完整的文件列表（包括不存在的），调用方自行过滤
func detectConfigFiles(configPath string) ([]ConfigFileEntry, error) {
	envPath, err := resolveEnvPath(configPath)
	if err != nil {
		return nil, err
	}

	// 用 godotenv.Read 读取 .env 值，不会污染当前进程环境变量
	envValues := make(map[string]string)
	if _, statErr := os.Stat(envPath); statErr == nil {
		vals, readErr := godotenv.Read(envPath)
		if readErr != nil {
			return nil, fmt.Errorf("读取 .env 文件失败 %s: %w", envPath, readErr)
		}
		envValues = vals
	} else if !os.IsNotExist(statErr) {
		return nil, fmt.Errorf("检查 .env 文件失败 %s: %w", envPath, statErr)
	}

	// 辅助函数：环境变量优先，其次 .env 中的值
	getVal := func(key string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return envValues[key]
	}

	sourceLabel := func(envKey, defaultDesc string) string {
		if os.Getenv(envKey) != "" {
			return envKey + " 环境变量"
		}
		if envValues[envKey] != "" {
			return envKey + "（.env 文件）"
		}
		return defaultDesc
	}

	var files []ConfigFileEntry

	// 1. .env 文件
	envSource := "默认路径"
	if configPath != "" {
		envSource = "--config 参数"
	}
	files = append(files, ConfigFileEntry{
		ArchiveName:  ".env",
		OriginalPath: envPath,
		Source:       envSource,
		Description:  "YewResin 配置文件",
	})

	// 2. rclone.conf
	rclonePath := getVal("RCLONE_CONFIG")
	rcloneSource := sourceLabel("RCLONE_CONFIG", "默认路径")
	if rclonePath == "" {
		rclonePath, err = defaultRcloneConfigPath()
		if err != nil {
			return nil, err
		}
		rcloneSource = "默认路径"
	}
	files = append(files, ConfigFileEntry{
		ArchiveName:  "rclone.conf",
		OriginalPath: rclonePath,
		Source:       rcloneSource,
		Description:  "Rclone 配置文件",
	})

	// 3. repository.config
	kopiaPath := getVal("KOPIA_CONFIG_FILE")
	kopiaSource := sourceLabel("KOPIA_CONFIG_FILE", "默认路径")
	if kopiaPath == "" {
		kopiaPath, err = defaultKopiaConfigPath()
		if err != nil {
			return nil, err
		}
		kopiaSource = "默认路径"
	}
	files = append(files, ConfigFileEntry{
		ArchiveName:  "repository.config",
		OriginalPath: kopiaPath,
		Source:       kopiaSource,
		Description:  "Kopia 仓库配置文件",
	})

	// 4. repository.config.kopia-password（同目录自动探测）
	kopiaPasswordPath := kopiaPath + ".kopia-password"
	files = append(files, ConfigFileEntry{
		ArchiveName:  "repository.config.kopia-password",
		OriginalPath: kopiaPasswordPath,
		Source:       "自动检测（同 repository.config 目录）",
		Description:  "Kopia 仓库密码文件",
	})

	return files, nil
}

// =================== 归档操作 ===================

func createTarGz(manifest *Manifest, files []ConfigFileEntry) (*bytes.Buffer, error) {
	var buf bytes.Buffer
	gw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gw)

	// 写入 manifest.json 作为第一个条目
	manifestJSON, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("序列化 manifest 失败: %w", err)
	}
	if err := tw.WriteHeader(&tar.Header{
		Name: "manifest.json",
		Size: int64(len(manifestJSON)),
		Mode: 0644,
	}); err != nil {
		return nil, fmt.Errorf("写入 manifest header 失败: %w", err)
	}
	if _, err := tw.Write(manifestJSON); err != nil {
		return nil, fmt.Errorf("写入 manifest 数据失败: %w", err)
	}

	// 写入每个配置文件
	for _, f := range files {
		data, err := os.ReadFile(f.OriginalPath)
		if err != nil {
			return nil, fmt.Errorf("读取文件 %s 失败: %w", f.OriginalPath, err)
		}
		info, err := os.Stat(f.OriginalPath)
		if err != nil {
			return nil, fmt.Errorf("获取文件信息 %s 失败: %w", f.OriginalPath, err)
		}
		if err := tw.WriteHeader(&tar.Header{
			Name:    f.ArchiveName,
			Size:    int64(len(data)),
			Mode:    int64(info.Mode()),
			ModTime: info.ModTime(),
		}); err != nil {
			return nil, fmt.Errorf("写入 tar header 失败: %w", err)
		}
		if _, err := tw.Write(data); err != nil {
			return nil, fmt.Errorf("写入 tar 数据失败: %w", err)
		}
	}

	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 tar writer 失败: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("关闭 gzip writer 失败: %w", err)
	}
	return &buf, nil
}

func extractTarGz(r io.Reader) (*Manifest, map[string][]byte, error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return nil, nil, fmt.Errorf("解压失败: %w", err)
	}
	defer gr.Close()

	tr := tar.NewReader(gr)
	fileContents := make(map[string][]byte)
	var manifest *Manifest

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("读取归档条目失败: %w", err)
		}

		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, nil, fmt.Errorf("读取归档数据失败: %w", err)
		}

		if header.Name == "manifest.json" {
			manifest = &Manifest{}
			if err := json.Unmarshal(data, manifest); err != nil {
				return nil, nil, fmt.Errorf("解析 manifest 失败: %w", err)
			}
		} else {
			fileContents[header.Name] = data
		}
	}

	if manifest == nil {
		return nil, nil, fmt.Errorf("归档中缺少 manifest.json")
	}
	return manifest, fileContents, nil
}

// =================== 加密/解密 ===================

func encryptWithAge(passphrase string, plaintext io.Reader, output io.Writer) error {
	recipient, err := age.NewScryptRecipient(passphrase)
	if err != nil {
		return fmt.Errorf("创建加密接收者失败: %w", err)
	}
	w, err := age.Encrypt(output, recipient)
	if err != nil {
		return fmt.Errorf("初始化加密失败: %w", err)
	}
	if _, err := io.Copy(w, plaintext); err != nil {
		return fmt.Errorf("加密数据写入失败: %w", err)
	}
	return w.Close()
}

func decryptWithAge(passphrase string, ciphertext io.Reader) (io.Reader, error) {
	identity, err := age.NewScryptIdentity(passphrase)
	if err != nil {
		return nil, fmt.Errorf("创建解密身份失败: %w", err)
	}
	r, err := age.Decrypt(ciphertext, identity)
	if err != nil {
		return nil, fmt.Errorf("解密失败（密码可能不正确）: %w", err)
	}
	return r, nil
}

// =================== 密码输入 ===================

func promptPassphrase(prompt string) (string, error) {
	fmt.Fprint(os.Stderr, prompt)
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Fprintln(os.Stderr) // 隐藏输入后换行
	if err != nil {
		return "", fmt.Errorf("读取密码失败: %w", err)
	}
	return string(password), nil
}

func promptPassphraseConfirm() (string, error) {
	pass1, err := promptPassphrase("输入加密密码: ")
	if err != nil {
		return "", err
	}
	if pass1 == "" {
		return "", fmt.Errorf("密码不能为空")
	}
	pass2, err := promptPassphrase("再次输入密码: ")
	if err != nil {
		return "", err
	}
	if pass1 != pass2 {
		return "", fmt.Errorf("两次输入的密码不一致")
	}
	return pass1, nil
}

// =================== 辅助函数 ===================

func humanSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

type ImportPlanEntry struct {
	ArchiveName string
	Description string
	SourcePath  string
	TargetPath  string
}

func detectConfigFileMap(configPath string) (map[string]ConfigFileEntry, error) {
	files, err := detectConfigFiles(configPath)
	if err != nil {
		return nil, err
	}
	fileMap := make(map[string]ConfigFileEntry, len(files))
	for _, file := range files {
		fileMap[file.ArchiveName] = file
	}
	return fileMap, nil
}

func resolveImportPlan(manifest *Manifest, configPath string) ([]ImportPlanEntry, error) {
	targets, err := detectConfigFileMap(configPath)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(manifest.Files))
	plan := make([]ImportPlanEntry, 0, len(manifest.Files))
	for _, file := range manifest.Files {
		if file.ArchiveName == "" {
			return nil, fmt.Errorf("manifest 包含空的归档文件名")
		}
		if _, exists := seen[file.ArchiveName]; exists {
			return nil, fmt.Errorf("manifest 中包含重复条目: %s", file.ArchiveName)
		}
		seen[file.ArchiveName] = struct{}{}

		target, ok := targets[file.ArchiveName]
		if !ok {
			return nil, fmt.Errorf("manifest 包含不受支持的配置文件: %s", file.ArchiveName)
		}

		targetPath := filepath.Clean(target.OriginalPath)
		if targetPath == "." || targetPath == "" {
			return nil, fmt.Errorf("无法确定 %s 的目标路径", target.Description)
		}

		plan = append(plan, ImportPlanEntry{
			ArchiveName: target.ArchiveName,
			Description: target.Description,
			SourcePath:  file.OriginalPath,
			TargetPath:  targetPath,
		})
	}

	return plan, nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("创建临时文件失败: %w", err)
	}

	tmpPath := tmpFile.Name()
	cleanup := true
	defer func() {
		if cleanup {
			tmpFile.Close()
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return fmt.Errorf("写入临时文件失败: %w", err)
	}
	if err := tmpFile.Sync(); err != nil {
		return fmt.Errorf("同步临时文件失败: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("关闭临时文件失败: %w", err)
	}
	if err := os.Chmod(tmpPath, 0600); err != nil {
		return fmt.Errorf("设置临时文件权限失败: %w", err)
	}

	if runtime.GOOS == "windows" {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("替换目标文件失败: %w", err)
		}
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("重命名临时文件失败: %w", err)
	}

	cleanup = false
	return nil
}

// =================== CLI 子命令 ===================

func printConfigUsage() {
	fmt.Fprintf(os.Stderr, "用法: %s config <子命令> [选项]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "子命令:\n")
	fmt.Fprintf(os.Stderr, "  export    导出配置文件（加密归档）\n")
	fmt.Fprintf(os.Stderr, "  import    导入配置文件（解密还原）\n")
	fmt.Fprintf(os.Stderr, "  list      列出可导出的配置文件\n")
}

func runConfigCmd(args []string) {
	if len(args) == 0 || isHelpFlag(args[0]) {
		printConfigUsage()
		if len(args) == 0 {
			os.Exit(1)
		}
		return
	}

	switch args[0] {
	case "export":
		runConfigExport(args[1:])
	case "import":
		runConfigImport(args[1:])
	case "list":
		runConfigList(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "未知的 config 子命令: %s\n\n", args[0])
		printConfigUsage()
		os.Exit(1)
	}
}

func runConfigList(args []string) {
	fs := flag.NewFlagSet("config list", flag.ExitOnError)
	configFile := fs.String("config", "", "配置文件路径")
	fs.Parse(args)

	files, err := detectConfigFiles(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "探测配置文件失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Println("YewResin 配置文件检测结果")
	fmt.Println("==========================================")

	found := 0
	for _, f := range files {
		fmt.Printf("\n  %s\n", f.Description)
		fmt.Printf("    路径: %s\n", f.OriginalPath)
		fmt.Printf("    来源: %s\n", f.Source)
		if info, err := os.Stat(f.OriginalPath); err == nil {
			fmt.Printf("    状态: 存在 (%s)\n", humanSize(info.Size()))
			found++
		} else {
			fmt.Printf("    状态: 不存在\n")
		}
	}

	fmt.Println()
	fmt.Println("==========================================")
	fmt.Printf("共检测 %d 个文件，其中 %d 个存在可导出\n", len(files), found)
	fmt.Println("==========================================")
}

func runConfigExport(args []string) {
	fs := flag.NewFlagSet("config export", flag.ExitOnError)
	outputFile := fs.String("output", "", "输出文件路径（默认: yewresin-config-YYYYMMDD.age）")
	fs.StringVar(outputFile, "o", "", "输出文件路径（简写）")
	configFile := fs.String("config", "", "配置文件路径")
	fs.Parse(args)

	// 探测配置文件
	files, err := detectConfigFiles(*configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "探测配置文件失败: %v\n", err)
		os.Exit(1)
	}

	// 过滤出存在的文件
	var existingFiles []ConfigFileEntry
	for _, f := range files {
		if _, err := os.Stat(f.OriginalPath); err == nil {
			existingFiles = append(existingFiles, f)
		}
	}
	if len(existingFiles) == 0 {
		fmt.Fprintln(os.Stderr, "未找到任何可导出的配置文件")
		os.Exit(1)
	}

	// 展示将导出的文件
	fmt.Println("\n将导出以下配置文件:")
	for _, f := range existingFiles {
		info, _ := os.Stat(f.OriginalPath)
		fmt.Printf("  %-35s %s (%s)\n", f.Description+":", f.OriginalPath, humanSize(info.Size()))
	}
	fmt.Println()

	// 提示输入密码
	passphrase, err := promptPassphraseConfirm()
	if err != nil {
		fmt.Fprintf(os.Stderr, "密码输入失败: %v\n", err)
		os.Exit(1)
	}

	// 构建 manifest
	manifest := &Manifest{
		Version:         1,
		CreatedAt:       time.Now().UTC().Format(time.RFC3339),
		YewResinVersion: version,
		Files:           existingFiles,
	}

	// 创建 tar.gz
	tarBuf, err := createTarGz(manifest, existingFiles)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建归档失败: %v\n", err)
		os.Exit(1)
	}

	// 确定输出路径
	if *outputFile == "" {
		*outputFile = fmt.Sprintf("yewresin-config-%s.age", time.Now().Format("20060102"))
	}

	// 加密并写入
	outFile, err := os.Create(*outputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "创建输出文件失败: %v\n", err)
		os.Exit(1)
	}
	defer outFile.Close()

	if err := encryptWithAge(passphrase, tarBuf, outFile); err != nil {
		os.Remove(*outputFile) // 清理不完整的文件
		fmt.Fprintf(os.Stderr, "加密失败: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("配置已导出到: %s\n", *outputFile)
}

func runConfigImport(args []string) {
	fs := flag.NewFlagSet("config import", flag.ExitOnError)
	force := fs.Bool("force", false, "强制覆盖已存在的文件")
	fs.BoolVar(force, "f", false, "强制覆盖（简写）")
	configFile := fs.String("config", "", "目标配置文件路径")
	fs.Parse(args)

	// 验证输入文件
	remaining := fs.Args()
	if len(remaining) == 0 {
		fmt.Fprintln(os.Stderr, "请指定要导入的归档文件路径")
		fmt.Fprintf(os.Stderr, "用法: %s config import [选项] <file.age>\n", os.Args[0])
		os.Exit(1)
	}
	inputFile := remaining[0]

	// 打开归档文件
	cipherData, err := os.Open(inputFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "打开归档文件失败: %v\n", err)
		os.Exit(1)
	}
	defer cipherData.Close()

	// 提示输入密码
	passphrase, err := promptPassphrase("输入解密密码: ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "密码输入失败: %v\n", err)
		os.Exit(1)
	}

	// 解密
	plainReader, err := decryptWithAge(passphrase, cipherData)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	// 解压并读取 manifest
	manifest, fileContents, err := extractTarGz(plainReader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析归档失败: %v\n", err)
		os.Exit(1)
	}

	importPlan, err := resolveImportPlan(manifest, *configFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成导入计划失败: %v\n", err)
		os.Exit(1)
	}

	// 展示归档信息
	fmt.Printf("\n归档信息: YewResin %s, 创建于 %s\n", manifest.YewResinVersion, manifest.CreatedAt)
	fmt.Println("\n将还原以下文件:")

	var conflicts []ImportPlanEntry
	for _, f := range importPlan {
		exists := false
		if _, statErr := os.Stat(f.TargetPath); statErr == nil {
			exists = true
			conflicts = append(conflicts, f)
		}
		status := "（新建）"
		if exists {
			status = "（已存在，将覆盖）"
		}
		fmt.Printf("  %-35s -> %s %s\n", f.Description, f.TargetPath, status)
		if f.SourcePath != "" && filepath.Clean(f.SourcePath) != f.TargetPath {
			fmt.Printf("    归档原路径: %s\n", f.SourcePath)
		}
	}

	// 确认覆盖
	if len(conflicts) > 0 && !*force {
		fmt.Printf("\n%d 个文件将被覆盖，确认继续？[y/N] ", len(conflicts))
		var response string
		fmt.Scanln(&response)
		if !strings.EqualFold(response, "y") && !strings.EqualFold(response, "yes") {
			fmt.Println("已取消导入")
			os.Exit(0)
		}
	}

	// 写入文件
	fmt.Println()
	restored := 0
	for _, f := range importPlan {
		data, ok := fileContents[f.ArchiveName]
		if !ok {
			fmt.Printf("  跳过: %s（归档中无数据）\n", f.Description)
			continue
		}

		// 创建父目录
		dir := filepath.Dir(f.TargetPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "  失败: 创建目录 %s: %v\n", dir, err)
			continue
		}

		if err := writeFileAtomic(f.TargetPath, data); err != nil {
			fmt.Fprintf(os.Stderr, "  失败: 写入 %s: %v\n", f.TargetPath, err)
			continue
		}
		fmt.Printf("  已还原: %s -> %s\n", f.Description, f.TargetPath)
		restored++
	}

	fmt.Printf("\n导入完成: %d/%d 个文件已还原\n", restored, len(importPlan))
}
