package email

import (
	"reflect"
	"testing"
)

func TestCollectRecipientsIncludesEnvelopeFallbackAndDeduplicates(t *testing.T) {
	to := `[{"Name":"订单","EmailAddress":"Order-123@shop.example.com"}]`
	cc := `[{"Name":"duplicate","EmailAddress":"order-123@SHOP.example.com"},{"Name":"抄送","EmailAddress":"cc@example.com"}]`
	bcc := `[{"Name":"","EmailAddress":"hidden@a.b.example.com"}]`

	got := collectRecipients(to, cc, bcc)
	want := []User{
		{Name: "订单", EmailAddress: "Order-123@shop.example.com"},
		{Name: "抄送", EmailAddress: "cc@example.com"},
		{Name: "", EmailAddress: "hidden@a.b.example.com"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("recipients = %#v, want %#v", got, want)
	}
}

func TestCollectRecipientsIgnoresInvalidAndEmptyValues(t *testing.T) {
	got := collectRecipients("not-json", `[{"Name":"","EmailAddress":""}]`, "")
	if len(got) != 0 {
		t.Fatalf("recipients = %#v, want empty", got)
	}
}
