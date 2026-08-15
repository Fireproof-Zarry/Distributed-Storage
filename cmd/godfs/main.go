package main

import (
	"flag"
	"fmt"
	"os"

	"godfs/internal/client"
	"godfs/internal/node"
	"godfs/internal/peer"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "node":
		runNode(os.Args[2:])
	case "upload":
		runUpload(os.Args[2:])
	case "download":
		runDownload(os.Args[2:])
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println("usage:")
	fmt.Println("  godfs node     -addr <host:port> -data <dir>")
	fmt.Println("  godfs upload   -file <path> -peers <peers.json> -manifest <out.json>")
	fmt.Println("  godfs download -manifest <path> -peers <peers.json> -out <path>")
}

func runNode(args []string) {
	fs := flag.NewFlagSet("node", flag.ExitOnError)
	addr := fs.String("addr", "localhost:8001", "address to listen on")
	dataDir := fs.String("data", "./data", "directory to store chunks")
	fs.Parse(args)

	n, err := node.NewNode(*addr, *dataDir)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
	if err := n.Listen(); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}
}

func runUpload(args []string) {
	fs := flag.NewFlagSet("upload", flag.ExitOnError)
	filePath := fs.String("file", "", "file to upload")
	peersPath := fs.String("peers", "./config/peers.json", "peer registry file")
	manifestPath := fs.String("manifest", "./manifest.json", "output path for the manifest")
	fs.Parse(args)

	if *filePath == "" {
		fmt.Println("error: -file is required")
		os.Exit(1)
	}

	registry, err := peer.LoadRegistry(*peersPath)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	m, err := client.Upload(*filePath, registry, *manifestPath)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Printf("\nupload complete: %s\n", *filePath)
	fmt.Printf("manifest saved to: %s\n", *manifestPath)
	fmt.Printf("manifest hash: %s\n", m.Hash())
}

func runDownload(args []string) {
	fs := flag.NewFlagSet("download", flag.ExitOnError)
	manifestPath := fs.String("manifest", "", "manifest file describing the upload")
	peersPath := fs.String("peers", "./config/peers.json", "peer registry file")
	outPath := fs.String("out", "./downloaded_output", "output file path")
	fs.Parse(args)

	if *manifestPath == "" {
		fmt.Println("error: -manifest is required")
		os.Exit(1)
	}

	registry, err := peer.LoadRegistry(*peersPath)
	if err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	if err := client.Download(*manifestPath, registry, *outPath); err != nil {
		fmt.Println("error:", err)
		os.Exit(1)
	}

	fmt.Printf("\ndownload complete: %s\n", *outPath)
}
