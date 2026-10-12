package guardrail

import (
	"slices"
	"testing"
)

func TestRedactMasksSensitiveValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		in        string
		want      string
		wantKinds []string
	}{
		{"masks_phone_with_spaces", "gọi mình 0901 234 567 nhé", "gọi mình [PHONE] nhé", []string{"PHONE"}},
		{"masks_phone_with_dot_groups", "sđt 0901.234.567", "sđt [PHONE]", []string{"PHONE"}},
		{"masks_phone_with_country_code", "+84 912 345 678", "[PHONE]", []string{"PHONE"}},
		{"masks_cccd", "CCCD của tôi 001203004567", "CCCD của tôi [CCCD]", []string{"CCCD"}},
		{"masks_old_cmnd_after_keyword", "CMND: 123456789", "CMND: [CCCD]", []string{"CCCD"}},
		{"masks_email", "mail a.b@example.com.vn", "mail [EMAIL]", []string{"EMAIL"}},
		{"masks_bank_account_after_keyword", "STK 1903 5678 9012 345 Techcombank", "STK [BANK_ACCOUNT] Techcombank", []string{"BANK_ACCOUNT"}},
		{"masks_card_passing_luhn", "thẻ 4111 1111 1111 1111", "thẻ [CARD]", []string{"CARD"}},
		{"masks_openrouter_key", "key sk-or-v1-abcdef0123456789abcdef", "key [API_KEY]", []string{"API_KEY"}},
		{"masks_aws_key", "AKIAIOSFODNN7EXAMPLE", "[API_KEY]", []string{"API_KEY"}},
		{"masks_jwt", "token eyJhbGciOi.eyJzdWIiOiIx.SflKxwRJSM", "token [TOKEN]", []string{"TOKEN"}},
		{"masks_password_value_only", "mật khẩu là Abc@2024", "mật khẩu là [PASSWORD]", []string{"PASSWORD"}},
		{"masks_password_with_equals", "password=hunter2", "password=[PASSWORD]", []string{"PASSWORD"}},
		{"masks_otp_after_keyword", "mã OTP 482913 đây", "mã OTP [OTP] đây", []string{"OTP"}},
		{"reports_each_kind_once", "0901234567 hoặc 0912345678", "[PHONE] hoặc [PHONE]", []string{"PHONE"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, kinds := Redact(tt.in)
			if got != tt.want || !slices.Equal(kinds, tt.wantKinds) {
				t.Errorf("Redact(%q) = %q, %v; want %q, %v", tt.in, got, kinds, tt.want, tt.wantKinds)
			}
		})
	}
}

func TestRedactLeavesOrdinaryTextAlone(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		in   string
	}{
		{"does_not_mask_revenue_figures", `{"ticker":"VNM","revenue":61782742000000,"netProfit":9019000000000,"year":2024}`},
		{"does_not_mask_volume_and_price", `{"symbol":"FPT","volume":12345678,"close":132500,"totalValue":1635802335000}`},
		{"does_not_mask_thousand_separated_numbers", "doanh thu 1.234.567.890 đồng, lợi nhuận 98,765,432,100"},
		{"does_not_mask_decimal_figures", "EPS 4111111111111111.25"},
		{"does_not_mask_card_failing_luhn", "4111 1111 1111 1112"},
		{"does_not_mask_account_number_without_keyword", "mã giao dịch 19035678901234"},
		{"does_not_mask_securities_account_phrase", "làm sao mở tài khoản chứng khoán ở SSI?"},
		{"does_not_mask_pass_in_english_prose", "will VNM pass the 80,000 level this year?"},
		{"does_not_mask_password_question", "mật khẩu là gì vậy, mình quên rồi"},
		{"does_not_mask_password_advice", "mật khẩu là không nên chia sẻ"},
		{"does_not_mask_ticker_and_year", "so sánh HPG 2023 và 2024"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, kinds := Redact(tt.in)
			if got != tt.in || len(kinds) != 0 {
				t.Errorf("Redact(%q) = %q, %v; want it unchanged", tt.in, got, kinds)
			}
		})
	}
}
