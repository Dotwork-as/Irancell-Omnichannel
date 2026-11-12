// Package i18n holds the backend's localization primitives: the
// supported locales, Accept-Language resolution and the fa/en message
// catalogs for API responses (docs/API.md §1 — backend-localized
// product content is chosen per locale by callers; these catalogs
// carry the shared strings, today the §4.3 error messages).
package i18n

import (
	"strconv"
	"strings"
)

// Supported locales (docs/API.md §1): fa is the default and fallback.
const (
	LocaleFa = "fa"
	LocaleEn = "en"
)

// Supported reports whether loc is a backend locale.
func Supported(loc string) bool {
	return loc == LocaleFa || loc == LocaleEn
}

// messages maps machine-readable error codes to their fa/en texts.
// The per-field VALIDATION_ERROR detail codes (INVALID_QUANTITY,
// VARIANT_NOT_FOUND, REQUIRED, NAME_LENGTH, ADDRESS_LENGTH,
// INVALID_PHONE, SHIPPING_METHOD_NOT_FOUND) reuse the same catalog
// via Message().
var messages = map[string]map[string]string{
	"FORBIDDEN_ORIGIN": {
		LocaleFa: "مبدأ درخواست مجاز نیست.",
		LocaleEn: "Request origin is not allowed.",
	},
	"NOT_FOUND": {
		LocaleFa: "منبع درخواست‌شده پیدا نشد.",
		LocaleEn: "The requested resource was not found.",
	},
	"METHOD_NOT_ALLOWED": {
		LocaleFa: "این روش درخواست برای این مسیر مجاز نیست.",
		LocaleEn: "This request method is not allowed for this route.",
	},
	"INTERNAL_ERROR": {
		LocaleFa: "خطای داخلی سرور رخ داد.",
		LocaleEn: "An internal server error occurred.",
	},
	"VALIDATION_ERROR": {
		LocaleFa: "درخواست نامعتبر است.",
		LocaleEn: "The request is invalid.",
	},
	"INVALID_REQUEST": {
		LocaleFa: "بدنهٔ درخواست قابل پردازش نیست.",
		LocaleEn: "The request body could not be processed.",
	},
	"INVALID_QUANTITY": {
		LocaleFa: "تعداد باید عددی بین ۱ تا ۹۹ باشد.",
		LocaleEn: "Quantity must be a number between 1 and 99.",
	},
	"VARIANT_NOT_FOUND": {
		LocaleFa: "این نسخهٔ محصول پیدا نشد.",
		LocaleEn: "This product variant was not found.",
	},
	"STOCK_UNAVAILABLE": {
		LocaleFa: "موجودی کافی نیست.",
		LocaleEn: "Not enough stock available.",
	},
	"CART_EMPTY": {
		LocaleFa: "سبد خرید خالی است.",
		LocaleEn: "The cart is empty.",
	},
	"REQUIRED": {
		LocaleFa: "این فیلد الزامی است.",
		LocaleEn: "This field is required.",
	},
	"NAME_LENGTH": {
		LocaleFa: "نام باید بین ۲ تا ۶۰ کاراکتر باشد.",
		LocaleEn: "Name must be between 2 and 60 characters.",
	},
	"ADDRESS_LENGTH": {
		LocaleFa: "آدرس باید بین ۱۰ تا ۵۰۰ کاراکتر باشد.",
		LocaleEn: "Address must be between 10 and 500 characters.",
	},
	"INVALID_PHONE": {
		LocaleFa: "شماره موبایل باید یک شمارهٔ ایرانسلی معتبر با ۱۱ رقم باشد (مثل ۰۹۱۲۳۴۵۶۷۸۹).",
		LocaleEn: "Phone must be a valid Iranian mobile number with 11 digits (09xxxxxxxxx).",
	},
	"SHIPPING_METHOD_NOT_FOUND": {
		LocaleFa: "شیوهٔ ارسال انتخابی معتبر نیست.",
		LocaleEn: "The selected shipping method is not valid.",
	},
	"RATE_LIMITED": {
		LocaleFa: "درخواست‌های شما بیش از حد مجاز است؛ کمی بعد دوباره تلاش کنید.",
		LocaleEn: "Too many requests. Please retry after a short while.",
	},
	"TURNSTILE_REQUIRED": {
		LocaleFa: "برای ادامهٔ پرداخت، تأیید اشخاص (Cloudflare Turnstile) لازم است.",
		LocaleEn: "Human verification (Cloudflare Turnstile) is required to continue checkout.",
	},
	"TURNSTILE_FAILED": {
		LocaleFa: "تأیید اشخاص ناموفق بود؛ دوباره تلاش کنید.",
		LocaleEn: "Human verification failed. Please try again.",
	},
}

// Message returns the localized text for a message code; unknown
// codes and unsupported locales fall back to the code itself and to
// Persian respectively.
func Message(locale, code string) string {
	msgs, ok := messages[code]
	if !ok {
		return code
	}
	if m, ok := msgs[locale]; ok && m != "" {
		return m
	}
	return msgs[LocaleFa]
}

// ResolveAcceptLanguage picks the highest-quality supported tag in an
// Accept-Language header. Tags match on the primary subtag (fa-IR →
// fa, en-US → en, case-insensitive); missing, unsupported or
// malformed headers fall back to fa.
func ResolveAcceptLanguage(header string) string {
	best := ""
	bestQ := -1.0

	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		tag := part
		q := 1.0
		if i := strings.IndexByte(part, ';'); i >= 0 {
			tag = strings.TrimSpace(part[:i])
			for _, p := range strings.Split(part[i+1:], ";") {
				p = strings.TrimSpace(p)
				if v, err := strconv.ParseFloat(strings.TrimPrefix(p, "q="), 64); err == nil {
					q = v
				}
			}
		}

		lang := primarySubtag(tag)
		if !Supported(lang) {
			continue
		}
		if q > bestQ {
			best, bestQ = lang, q
		}
	}

	if best == "" {
		return LocaleFa
	}
	return best
}

func primarySubtag(tag string) string {
	if i := strings.IndexAny(tag, "-_"); i > 0 {
		return strings.ToLower(tag[:i])
	}
	return strings.ToLower(tag)
}
