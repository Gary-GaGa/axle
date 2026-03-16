package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/garyellow/axle/internal/foodsafety"
)

func main() {
	days := flag.Int("days", 7, "查詢最近 N 天的資料（0 = 全部）")
	types := flag.String("type", "", "查詢類型，逗號分隔 (violation,border,poison,alert)；留空查詢全部")
	summary := flag.Bool("summary", false, "只顯示摘要統計")
	limit := flag.Int("limit", 5, "每個類別最多顯示幾筆")
	timeout := flag.Int("timeout", 60, "API 超時秒數")
	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeout)*time.Second)
	defer cancel()

	var typeList []string
	if *types != "" {
		typeList = strings.Split(*types, ",")
	}

	client := foodsafety.NewClient()
	results := client.FetchAll(ctx, typeList)

	var output string
	if *summary {
		output = foodsafety.FormatSummary(results, *days)
	} else {
		output = foodsafety.FormatResults(results, *days, *limit)
	}

	fmt.Println(output)

	// 若有任何查詢失敗，以非零退出碼結束
	for _, r := range results {
		if r.Err != nil {
			os.Exit(1)
		}
	}
}
