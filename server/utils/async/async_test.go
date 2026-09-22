package async

import (
	"testing"

	"github.com/Jinnrry/pmail/utils/context"
)

func TestWaitIncludesConcurrentPanicRecovery(t *testing.T) {
	as := New(&context.Context{})
	for i := 0; i < 20; i++ {
		as.WaitProcess(func(any) { panic("expected test failure") }, nil)
	}
	as.Wait()
	if as.LastError() == nil {
		t.Fatal("Wait returned before recording recovered failure")
	}
}
