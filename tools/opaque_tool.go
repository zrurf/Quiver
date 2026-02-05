package main

import (
	"crypto"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/bytemare/ksf"
	"github.com/bytemare/opaque"
)

func main() {
	var (
		outputFormat = flag.String("format", "bin", "输出格式: bin, base64")
		outputDir    = flag.String("dir", "./secrets", "输出目录（bin格式使用）")
		verifyOnly   = flag.Bool("verify", false, "验证现有密钥文件")
		skipKeyCheck = flag.Bool("skip-check", false, "跳过公私钥校验（仅生成时有效）")
		showHelp     = flag.Bool("h", false, "显示帮助信息")
	)
	flag.Parse()

	if *showHelp {
		printUsage()
		return
	}

	fmt.Println("OPAQUE RistrettoSha512 密钥工具")
	fmt.Println("==================================")

	if *verifyOnly {
		verifyKeys(*outputDir, !*skipKeyCheck)
		return
	}

	// 生成密钥
	conf := &opaque.Configuration{
		OPRF:    opaque.RistrettoSha512,
		KDF:     crypto.SHA512,
		MAC:     crypto.SHA512,
		Hash:    crypto.SHA512,
		KSF:     ksf.Argon2id,
		AKE:     opaque.RistrettoSha512,
		Context: nil,
	}
	generateKeys(conf, *outputFormat, *outputDir, !*skipKeyCheck)
}

func printUsage() {
	fmt.Println(`用法: opaque_key_tool [选项]

选项:
  -format string     输出格式: bin, base64 (默认 "bin")
  -dir string        输出目录（bin格式使用） (默认 "./secrets")
  -verify            验证现有密钥文件
  -skip-check        跳过公私钥校验（仅生成时有效）
  -h                 显示此帮助信息

示例:
  # 生成并校验密钥文件
  opaque_key_tool -format bin -dir ./secrets

  # 生成Base64输出并校验
  opaque_key_tool -format base64

  # 仅生成不校验
  opaque_key_tool -format bin -skip-check

  # 验证密钥文件（包含公私钥校验）
  opaque_key_tool -verify -dir ./secrets

  # 验证密钥文件（仅检查文件存在和长度）
  opaque_key_tool -verify -dir ./secrets -skip-check`)
}

func generateKeys(conf *opaque.Configuration, format, dir string, doKeyCheck bool) {
	fmt.Println("生成 RistrettoSha512 密钥材料...")

	// 1. 生成 OPRF 种子
	oprfSeed := make([]byte, 64)
	if _, err := rand.Read(oprfSeed); err != nil {
		fatalErr("生成 OPRF 种子失败", err)
	}

	// 2. 生成服务器密钥对
	serverSecretKey, serverPublicKey := conf.KeyGen()

	fmt.Printf("✓ OPRF种子: %d 字节\n", len(oprfSeed))
	fmt.Printf("✓ 服务器公钥: %d 字节\n", len(serverPublicKey))
	fmt.Printf("✓ 服务器私钥: %d 字节\n", len(serverSecretKey))

	// 3. 校验公私钥
	if doKeyCheck {
		fmt.Println("正在进行公私钥校验...")
		if err := verifyKeyMaterial(conf, oprfSeed, serverPublicKey, serverSecretKey); err != nil {
			fatalErr("公私钥校验失败", err)
		}
		fmt.Println("✓ 公私钥校验通过")
	} else {
		fmt.Println("跳过公私钥校验")
	}

	// 4. 输出
	switch format {
	case "bin":
		saveBinaryFiles(dir, oprfSeed, serverPublicKey, serverSecretKey, doKeyCheck)
	case "base64":
		printBase64(oprfSeed, serverPublicKey, serverSecretKey)
	default:
		fmt.Printf("未知格式: %s，使用默认bin格式\n", format)
		saveBinaryFiles(dir, oprfSeed, serverPublicKey, serverSecretKey, doKeyCheck)
	}
}

func saveBinaryFiles(dir string, oprfSeed, serverPubKey, serverSecretKey []byte, doKeyCheck bool) {
	if err := os.MkdirAll(dir, 0700); err != nil {
		fatalErr("创建目录失败", err)
	}

	// 保存原始二进制文件
	files := []struct {
		path string
		data []byte
		perm os.FileMode
	}{
		{filepath.Join(dir, "oprf_seed.bin"), oprfSeed, 0600},
		{filepath.Join(dir, "server_public.bin"), serverPubKey, 0644},
		{filepath.Join(dir, "server_secret.bin"), serverSecretKey, 0600},
	}

	for _, file := range files {
		if err := os.WriteFile(file.path, file.data, file.perm); err != nil {
			fatalErr(fmt.Sprintf("写入文件 %s 失败", file.path), err)
		}
		fmt.Printf("✓ 已保存: %s (%d bytes)\n", file.path, len(file.data))
	}

	// 同时生成对应的Base64文件（便于查看）
	saveBase64Files(dir, oprfSeed, serverPubKey, serverSecretKey)

	if doKeyCheck {
		fmt.Println("\n✓ 密钥生成完成，已通过完整性校验")
	} else {
		fmt.Println("\n✓ 密钥生成完成（未校验完整性）")
	}
}

func saveBase64Files(dir string, oprfSeed, serverPubKey, serverSecretKey []byte) {
	files := []struct {
		path string
		data []byte
		perm os.FileMode
	}{
		{filepath.Join(dir, "oprf_seed.b64"), []byte(base64.StdEncoding.EncodeToString(oprfSeed)), 0600},
		{filepath.Join(dir, "server_public.b64"), []byte(base64.StdEncoding.EncodeToString(serverPubKey)), 0644},
		{filepath.Join(dir, "server_secret.b64"), []byte(base64.StdEncoding.EncodeToString(serverSecretKey)), 0600},
	}

	for _, file := range files {
		if err := os.WriteFile(file.path, file.data, file.perm); err != nil {
			fatalErr(fmt.Sprintf("写入Base64文件 %s 失败", file.path), err)
		}
		fmt.Printf("✓ Base64文件: %s\n", file.path)
	}
}

func printBase64(oprfSeed, serverPubKey, serverSecretKey []byte) {
	fmt.Println("\n=== Base64 输出 ===")
	fmt.Printf("OPRF_SEED=%s\n", base64.StdEncoding.EncodeToString(oprfSeed))
	fmt.Printf("SERVER_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(serverPubKey))
	fmt.Printf("SERVER_SECRET_KEY=%s\n", base64.StdEncoding.EncodeToString(serverSecretKey))

	fmt.Println("\n=== 原始字节长度 ===")
	fmt.Printf("OPRF种子: %d bytes\n", len(oprfSeed))
	fmt.Printf("服务器公钥: %d bytes\n", len(serverPubKey))
	fmt.Printf("服务器私钥: %d bytes\n", len(serverSecretKey))

	fmt.Println("\n✓ 密钥生成完成")
}

func verifyKeys(dir string, doKeyCheck bool) {
	fmt.Println("验证密钥文件...")

	// 检查文件存在和长度
	files := []struct {
		name   string
		desc   string
		binLen int
	}{
		{"oprf_seed.bin", "OPRF种子", 64},
		{"server_public.bin", "服务器公钥", 32},
		{"server_secret.bin", "服务器私钥", 32},
	}

	keyData := make(map[string][]byte)
	allValid := true

	for _, file := range files {
		path := filepath.Join(dir, file.name)
		data, err := os.ReadFile(path)
		if err != nil {
			fmt.Printf("✗ 无法读取 %s: %v\n", file.desc, err)
			allValid = false
			continue
		}

		if len(data) != file.binLen {
			fmt.Printf("✗ %s 长度错误: 期望 %d, 实际 %d\n", file.desc, file.binLen, len(data))
			allValid = false
		} else {
			fmt.Printf("✓ %s 验证通过: %d bytes\n", file.desc, len(data))
			keyData[file.name] = data

			if len(data) >= 4 {
				fmt.Printf("  起始字节: %02x %02x %02x %02x...\n",
					data[0], data[1], data[2], data[3])
			}
		}
	}

	if !allValid {
		fmt.Println("\n✗ 基础文件验证失败，跳过校验。")
		os.Exit(1)
	}

	// 验证Base64文件一致性
	fmt.Println("\n验证Base64文件一致性...")
	verifyBase64Consistency(dir)

	// 校验公私钥匹配性
	if doKeyCheck && allValid {
		fmt.Println("\n正在进行公私钥校验...")
		conf := &opaque.Configuration{
			OPRF:    opaque.RistrettoSha512,
			KDF:     crypto.SHA512,
			MAC:     crypto.SHA512,
			Hash:    crypto.SHA512,
			KSF:     ksf.Argon2id,
			AKE:     opaque.RistrettoSha512,
			Context: nil,
		}

		oprfSeed := keyData["oprf_seed.bin"]
		serverPubKey := keyData["server_public.bin"]
		serverSecretKey := keyData["server_secret.bin"]

		if err := verifyKeyMaterial(conf, oprfSeed, serverPubKey, serverSecretKey); err != nil {
			fmt.Printf("✗ 公私钥校验失败: %v\n", err)
			allValid = false
		} else {
			fmt.Println("✓ 公私钥校验通过")
		}
	} else if allValid {
		fmt.Println(" 跳过公私钥校验")
	}

	if allValid {
		fmt.Println("\n✓ 所有密钥文件验证通过！")
	} else {
		fmt.Println("\n✗ 密钥文件验证失败。")
		os.Exit(1)
	}
}

func verifyKeyMaterial(conf *opaque.Configuration, oprfSeed, serverPubKey, serverSecretKey []byte) error {
	// 创建服务器实例
	server, err := opaque.NewServer(conf)
	if err != nil {
		return fmt.Errorf("创建服务器实例失败: %w", err)
	}

	// 尝试设置密钥材料，这会验证密钥对的有效性
	if err := server.SetKeyMaterial(nil, serverSecretKey, serverPubKey, oprfSeed); err != nil {
		return fmt.Errorf("设置密钥材料失败: %w", err)
	}

	return nil
}

func verifyBase64Consistency(dir string) {
	binFiles := []string{"oprf_seed.bin", "server_public.bin", "server_secret.bin"}
	b64Files := []string{"oprf_seed.b64", "server_public.b64", "server_secret.b64"}

	for i := 0; i < len(binFiles); i++ {
		binPath := filepath.Join(dir, binFiles[i])
		b64Path := filepath.Join(dir, b64Files[i])

		binData, err1 := os.ReadFile(binPath)
		b64Data, err2 := os.ReadFile(b64Path)

		if err2 != nil {
			// Base64文件不存在是正常的
			continue
		}

		if err1 != nil {
			fmt.Printf("✗ 无法读取 %s: %v\n", binFiles[i], err1)
			continue
		}

		// 解码Base64并与二进制比较
		decoded, err := base64.StdEncoding.DecodeString(string(b64Data))
		if err != nil {
			fmt.Printf("✗ %s Base64解码失败: %v\n", b64Files[i], err)
			continue
		}

		if hex.EncodeToString(binData) != hex.EncodeToString(decoded) {
			fmt.Printf("✗ %s 与 %s 内容不一致\n", binFiles[i], b64Files[i])
		} else {
			fmt.Printf("✓ %s ↔ %s 一致\n", binFiles[i], b64Files[i])
		}
	}
}

func fatalErr(msg string, err error) {
	fmt.Printf("✗ %s: %v\n", msg, err)
	os.Exit(1)
}
