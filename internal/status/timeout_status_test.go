package status_test

import (
	"migpt-go/internal/status"
	"sync"
	"testing"
	"time"
)

func TestTimeoutStatus(t *testing.T) {
	t.Run("Add & Ok (正确版本)", func(t *testing.T) {
		ts := status.NewTimeoutStatus(time.Second)

		key := "test1"
		added, err := ts.Add(key, nil)
		if !added || err != nil {
			t.Fatalf("Add failed: %v", err)
		}

		ok, err := ts.Ok(key)
		if err != nil || ok {
			t.Errorf("Initial state should be false, got: %v", ok)
		}
	})
}

// 完整测试用例参考以下代码：

func TestTimeoutStatusLifecycle(t *testing.T) {
	const timeout = 100 * time.Millisecond
	ts := status.NewTimeoutStatus(timeout)
	key := "session1"

	// 添加新 key
	if added, err := ts.Add(key, nil); !added || err != nil {
		t.Fatalf("Add failed: %v", err)
	}

	// 续期不存在 key 应报错
	if _, err := ts.Renewal("invalid_key"); err == nil {
		t.Error("Expected error for invalid key renewal")
	}

	// --- Start 测试 ---
	exitNotify := make(chan struct{})
	ts = status.NewTimeoutStatus(timeout) // 重置
	// 正确添加并设置退出回调
	ts.Add(key, func() { close(exitNotify) })

	started, err := ts.Start(key)
	if !started || err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	// 初始状态应为 true（激活）
	if ok, _ := ts.Ok(key); !ok {
		t.Error("State should be active after Start")
	}

	// 等待超时触发
	select {
	case <-exitNotify:
		// 超时后状态应为 false
		if ok, _ := ts.Ok(key); ok {
			t.Error("State should be inactive after timeout")
		}
	case <-time.After(2 * timeout):
		t.Error("Timeout did not trigger")
	}

	// --- Renewal 测试 ---
	ts = status.NewTimeoutStatus(timeout) // 重置
	ts.Add(key, nil)
	ts.Start(key)

	// 提前续期
	time.Sleep(50 * time.Millisecond) // 未超时前
	ts.Renewal(key)

	// 验证续期生效（等待原超时时间）
	time.Sleep(50 * time.Millisecond)
	if ok, _ := ts.Ok(key); !ok {
		t.Error("Renewal failed, state expired")
	}
}

// 测试并发重置的安全性
func TestConcurrentRenewal(t *testing.T) {
	const (
		key     = "concurrent_key"
		timeout = time.Second
		workers = 10
	)

	ts := status.NewTimeoutStatus(timeout)
	ts.Add(key, nil)
	ts.Start(key)

	var wg sync.WaitGroup
	wg.Add(workers)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			ts.Renewal(key)
		}()
	}
	wg.Wait()

	// 最终应只重置了一次计时器
	if ok, _ := ts.Ok(key); !ok {
		t.Error("Concurrent renewal failed")
	}
}
