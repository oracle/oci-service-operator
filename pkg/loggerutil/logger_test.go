package loggerutil

import (
	"context"
	"errors"
	"github.com/stretchr/testify/assert"
	ctrl "sigs.k8s.io/controller-runtime"
	"strconv"
	"sync"
	"testing"
)

func Test_ConcurrentLoggerUpdating(t *testing.T) {
	testRoutineNum := 20
	var wg sync.WaitGroup
	wg.Add(testRoutineNum)

	logger := OSOKLogger{Logger: ctrl.Log.WithName("service-manager").WithName("VD")}

	for i := 0; i < testRoutineNum; i++ {
		go func(i int) {
			defer wg.Done()
			ctx := context.Background()
			fixedLogMap := make(map[string]string)
			fixedLogMap["index"] = strconv.Itoa(i)
			ctx = context.WithValue(ctx, FixedLogMapCtxKey, fixedLogMap)
			assert.NotPanics(t, func() {
				logger.InfoLogWithFixedMessage(ctx, "test concurrent info log")
				logger.ErrorLogWithFixedMessage(ctx, errors.New("test error"), "test concurrent error log")
				logger.DebugLogWithFixedMessage(ctx, "test concurrent debug log")
			})
		}(i)
	}

	wg.Wait()
}
