package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	var protoFiles []string

	// Scan all .proto files in proto/ directory
	err := filepath.Walk("proto", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Ext(path) == ".proto" {
			protoFiles = append(protoFiles, path)
		}
		return nil
	})

	if err != nil {
		fmt.Printf("❌ Failed to scan proto directory: %v\n", err)
		os.Exit(1)
	}

	if len(protoFiles) == 0 {
		fmt.Println("⚠️ No .proto files found in proto/ directory")
		os.Exit(1)
	}

	fmt.Printf("🔍 Found %d proto files. Compiling...\n", len(protoFiles))
	for _, f := range protoFiles {
		fmt.Printf("   📄 %s\n", f)
	}

	// Configure protoc args using module name 'go-boilerplate'
	args := []string{
		"--proto_path=.",
		"--go_out=.",
		"--go_opt=module=go-boilerplate",
		"--go-grpc_out=.",
		"--go-grpc_opt=module=go-boilerplate",
	}
	args = append(args, protoFiles...)

	cmd := exec.Command("protoc", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("\n❌ Protoc execution failed: %v\n", err)
		fmt.Println("💡 Hint: Ensure 'protoc', 'protoc-gen-go', and 'protoc-gen-go-grpc' are installed and added to system PATH.")
		os.Exit(1)
	}

	fmt.Println("\n✅ Protobuf compilation succeeded!")
}
