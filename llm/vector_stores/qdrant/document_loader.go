package qdrant_store

import (
	"context"
	"fmt"
	"migpt-go/config"
	"net/url"
	"os"
	"strings"

	"github.com/tmc/langchaingo/documentloaders"
	"github.com/tmc/langchaingo/textsplitter"
	"github.com/tmc/langchaingo/vectorstores/qdrant"
)

const (
	DocumentCollection = "DocumentCollection"
)

func LoadDocument(ctx context.Context, path string) error {
	u, err := url.Parse(config.DefaultConfig.Ai.Qdrant.Url)
	if err != nil {
		return err
	}

	q, err := qdrant.New(
		qdrant.WithURL(*u),
	)

	if err != nil {
		return err
	}

	dirs, err := os.ReadDir(path)
	if err != nil {
		return fmt.Errorf("read directory error: %w", err)
	}

	for _, dir := range dirs {
		if dir.IsDir() {
			continue
		}

		info, err := dir.Info()
		if err != nil {
			continue
		}

		filePath := path + "/" + dir.Name()
		file, err := os.Open(filePath)
		if err != nil {
			return fmt.Errorf("open file error %s: %w", filePath, err)
		}
		defer file.Close()

		var loader documentloaders.Loader

		// 根据文件扩展名选择合适的加载器
		switch getFileExtension(dir.Name()) {
		case ".csv":
			loader = documentloaders.NewCSV(file)
		case ".html", ".htm":
			loader = documentloaders.NewHTML(file)
		case ".txt":
			loader = documentloaders.NewText(file)
		case ".pdf":
			loader = documentloaders.NewPDF(file, info.Size())
		default:
			fmt.Printf("Unsupported file format: %s\n", dir.Name())
			continue
		}

		splitter := textsplitter.NewRecursiveCharacter()
		splitter.ChunkSize = 200
		splitter.ChunkOverlap = 30
		// 处理加载的文档

		docs, err := loader.LoadAndSplit(ctx, splitter)
		if err != nil {
			return err
		}

		q.AddDocuments(ctx, docs)
	}

	return nil
}

// getFileExtension 获取文件扩展名（包含点号）
func getFileExtension(filename string) string {
	for i := len(filename) - 1; i >= 0; i-- {
		if filename[i] == '.' {
			return strings.ToLower(filename[i:])
		}
	}
	return ""
}
