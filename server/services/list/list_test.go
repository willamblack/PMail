package list

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Jinnrry/pmail/dto"
	"github.com/Jinnrry/pmail/utils/context"
	_ "modernc.org/sqlite"
	"xorm.io/xorm"
)

func TestCountQueryIsNotPaginated(t *testing.T) {
	ctx := &context.Context{UserID: 1}
	tag := dto.SearchTag{Type: -1, Status: -1, GroupId: -1}
	listSQL, _ := genSQL(ctx, false, tag, "", "all", false, 15, 15)
	countSQL, countParams := genSQL(ctx, true, tag, "", "all", false, 15, 15)
	if !strings.Contains(listSQL, "LIMIT 15 OFFSET 15") {
		t.Fatalf("page 2 query must be paginated: %s", listSQL)
	}
	if strings.Contains(strings.ToUpper(countSQL), "LIMIT") || strings.Contains(strings.ToUpper(countSQL), "OFFSET") {
		t.Fatalf("count query must not be paginated: %s", countSQL)
	}

	engine, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	engine.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = engine.Close() })
	if _, err = engine.Exec(`create table email (id integer primary key)`); err != nil {
		t.Fatalf("create email table: %v", err)
	}
	if _, err = engine.Exec(`create table user_email (email_id integer, user_id integer, status integer, group_id integer)`); err != nil {
		t.Fatalf("create user_email table: %v", err)
	}
	for id := 1; id <= 31; id++ {
		if _, err = engine.Exec(`insert into email (id) values (?)`, id); err != nil {
			t.Fatalf("insert email %d: %v", id, err)
		}
		if _, err = engine.Exec(`insert into user_email (email_id, user_id, status, group_id) values (?, 1, 0, 0)`, id); err != nil {
			t.Fatalf("insert user_email %d: %v", id, err)
		}
	}
	var total int64
	if _, err = engine.SQL(countSQL, countParams...).Get(&total); err != nil {
		t.Fatalf("count page 2: %v", err)
	}
	if total != 31 {
		t.Fatalf("page 2 total = %d, want 31", total)
	}
}

func TestAppendKeywordSearchRecipient(t *testing.T) {
	sql, params := appendKeywordSearch("select * from email e where 1=1", []any{7}, "order-123@shop.example.com", "recipient", quoteTestColumn)

	for _, column := range []string{`e."to"`, `e."cc"`, `e."bcc"`} {
		if !strings.Contains(sql, column+" like ?") {
			t.Errorf("recipient search SQL does not contain %s: %s", column, sql)
		}
	}
	for _, column := range []string{`e."subject"`, `e."text"`, `e."from_address"`} {
		if strings.Contains(sql, column+" like ?") {
			t.Errorf("recipient-only search unexpectedly contains %s: %s", column, sql)
		}
	}
	wantParams := []any{7, "%order-123@shop.example.com%", "%order-123@shop.example.com%", "%order-123@shop.example.com%"}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("params = %#v, want %#v", params, wantParams)
	}
}

func TestAppendKeywordSearchAllFieldsAndRejectsFieldInjection(t *testing.T) {
	const untrustedField = "recipient); drop table email; --"
	sql, params := appendKeywordSearch("select * from email e where 1=1", nil, "测试123", untrustedField, quoteTestColumn)

	for _, column := range []string{`e."subject"`, `e."text"`, `e."html"`, `e."from_address"`, `e."from_name"`, `e."to"`, `e."cc"`, `e."bcc"`} {
		if !strings.Contains(sql, column+" like ?") {
			t.Errorf("all-field search SQL does not contain %s: %s", column, sql)
		}
	}
	if strings.Contains(sql, untrustedField) {
		t.Fatalf("untrusted search field was interpolated into SQL: %s", sql)
	}
	if len(params) != 8 {
		t.Fatalf("parameter count = %d, want 8", len(params))
	}
	for _, param := range params {
		if param != "%测试123%" {
			t.Fatalf("parameter = %#v, want %%测试123%%", param)
		}
	}
}

func TestAppendKeywordSearchIgnoresWhitespace(t *testing.T) {
	const originalSQL = "select * from email e"
	originalParams := []any{1}
	sql, params := appendKeywordSearch(originalSQL, originalParams, "   ", "recipient", quoteTestColumn)
	if sql != originalSQL || !reflect.DeepEqual(params, originalParams) {
		t.Fatalf("whitespace search changed query: sql=%q params=%#v", sql, params)
	}
}

func quoteTestColumn(column string) string {
	return `"` + column + `"`
}

func TestRecipientSearchRunsOnSQLite(t *testing.T) {
	engine, err := xorm.NewEngine("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open SQLite: %v", err)
	}
	t.Cleanup(func() { _ = engine.Close() })

	if _, err = engine.Exec(`create table email ("to" text, cc text, bcc text)`); err != nil {
		t.Fatalf("create email table: %v", err)
	}
	if _, err = engine.Exec(`insert into email ("to", cc, bcc) values (?, ?, ?)`,
		`[{"EmailAddress":"alias@a.b.example.com"}]`, "[]", "[]"); err != nil {
		t.Fatalf("insert email: %v", err)
	}

	query, params := appendKeywordSearch("select count(*) from email e where 1=1", nil, "alias@a.b.example.com", "recipient", engine.Quote)
	var count int64
	if _, err = engine.SQL(query, params...).Get(&count); err != nil {
		t.Fatalf("run recipient search: %v; SQL: %s", err, query)
	}
	if count != 1 {
		t.Fatalf("recipient search count = %d, want 1", count)
	}
}
