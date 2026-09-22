package imap_server

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/Jinnrry/pmail/utils/context"
	"github.com/emersion/go-imap/v2/imapserver"
)

func TestIdleImmediateReentry(t *testing.T) {
	conn, err := tls.Dial("tcp", imapTestAddr, &tls.Config{InsecureSkipVerify: true})
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err = conn.SetDeadline(time.Now().Add(10 * time.Second)); err != nil {
		t.Fatal(err)
	}
	reader := bufio.NewReader(conn)
	if _, err = reader.ReadString('\n'); err != nil {
		t.Fatal(err)
	}
	for _, command := range []struct{ tag, text string }{{"login", "LOGIN testCase testCase"}, {"select", "SELECT INBOX"}} {
		if _, err = fmt.Fprintf(conn, "%s %s\r\n", command.tag, command.text); err != nil {
			t.Fatal(err)
		}
		if line, readErr := readTaggedLine(reader, command.tag); readErr != nil || !strings.Contains(line, " OK") {
			t.Fatalf("command failed: %q, %v", line, readErr)
		}
	}
	for cycle := 0; cycle < 25; cycle++ {
		tag := fmt.Sprintf("idle%d", cycle)
		if _, err = fmt.Fprintf(conn, "%s IDLE\r\n", tag); err != nil {
			t.Fatal(err)
		}
		if line, readErr := reader.ReadString('\n'); readErr != nil || !strings.HasPrefix(line, "+") {
			t.Fatalf("IDLE continuation failed: %q, %v", line, readErr)
		}
		if _, err = conn.Write([]byte("DONE\r\n")); err != nil {
			t.Fatal(err)
		}
		if line, readErr := readTaggedLine(reader, tag); readErr != nil || !strings.Contains(line, " OK") {
			t.Fatalf("DONE failed in cycle %d: %q, %v", cycle, line, readErr)
		}
	}
}

// go-imap waits for Idle to return before acknowledging DONE. Returning before
// cleanup lets a previous IDLE's cleanup delete the next IDLE's registration.
func TestIdleLifetimeEndsOnlyAfterCleanup(t *testing.T) {
	const userID = 900001
	ctx := &context.Context{UserID: userID}
	ctx.SetValue(context.LogID, "idle-lifecycle")
	session := &serverSession{ctx: ctx, currentMailbox: "INBOX"}
	for cycle := 0; cycle < 3; cycle++ {
		stop := make(chan struct{})
		done := make(chan error, 1)
		go func() { done <- session.Idle(&imapserver.UpdateWriter{}, stop) }()
		select {
		case err := <-done:
			close(stop)
			waitIdleConnectionsDeleted(t, userID)
			t.Fatalf("Idle returned before stop in cycle %d: %v", cycle, err)
		case <-time.After(20 * time.Millisecond):
		}
		if _, ok := userConnects.Load(userID); !ok {
			close(stop)
			<-done
			t.Fatal("active Idle is not registered")
		}
		close(stop)
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("Idle did not stop")
		}
		if _, ok := userConnects.Load(userID); ok {
			t.Fatal("Idle returned before registration cleanup")
		}
	}
}
