package actions

import (
	"encoding/json"
	"log"
	"time"
	"fmt"
	"sync"
	"github.com/totooicu/go-mytool/stream"
)

	var WaitGroup sync.WaitGroup

func callCalculator() {
    targetStream := stream.Custom["calculator-stream"].(string)
    nums := [1000000]float64{}
    for i := range nums { nums[i] = 1.0 }
    payload := map[string]interface{}{"nums": nums}

    // 1. 单次请求耗时（预热）
    _, err := stream.Send(targetStream, "add", payload, 0)
    if err != nil {
        log.Fatal("single request failed:", err)
    }

    const groupNum = 10

    // 2. 串行测试：执行 10 次，记录总耗时
    start := time.Now()
    for i := 0; i < groupNum; i++ {
        _, err := stream.Send(targetStream, "add", payload, 0)
        if err != nil {
            log.Printf("serial request %d failed: %v", i, err)
        }
    }
    serialTotal := time.Since(start)
    fmt.Printf("串行总耗时: %v\n", serialTotal)

    // 3. 并发测试：使用 WaitGroup 真正并发执行 10 次
    var wg sync.WaitGroup
    wg.Add(groupNum)
    start = time.Now()
    for i := 0; i < groupNum; i++ {
        go func(id int) {
            defer wg.Done()
            _, err := stream.Send(targetStream, "add", payload, 0)
            if err != nil {
                log.Printf("concurrent request %d failed: %v", id, err)
            }
        }(i)
    }
    wg.Wait()
    concurrentTotal := time.Since(start)
    fmt.Printf("并发总耗时: %v\n", concurrentTotal)
}

func interfaceToBytes(data map[string]interface{}) []byte {
	b, _ := json.Marshal(data)
	return b
}