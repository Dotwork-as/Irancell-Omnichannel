// Package seed applies the idempotent startup seeds of the backend:
// the six catalog products with variants and localized content, the
// two shipping methods, and (optionally) demo orders. All content
// mirrors the frontend catalogs (messages/{fa,en}.json); product
// slugs are PROVISIONAL pending the OPEN decision in
// docs/API.md §8 — change them in one place here.
package seed

// Storage-tier pricing rule (provisional merchandising rule, easy to
// change): the frontend catalog quotes one price per product for its
// default storage tier; every tier above the default adds
// storageTierStepIrr, every tier below subtracts it. A product's
// compare-at gap (frontend oldPrice − price) stays constant across
// tiers. Prices are integer IRR = frontend toman × 10.
const storageTierStepIrr = 25_000_000

type colorSeed struct {
	Key    string
	NameFa string
	NameEn string
}

type variantSeed struct {
	Color      colorSeed
	StorageGb  int
	TierOffset int // storage tiers relative to the product's default tier
	StockQty   int
	Sku        string
}

type specSeed struct {
	LabelFa string
	LabelEn string
	ValueFa string
	ValueEn string
}

type reviewSeed struct {
	AuthorFa string
	AuthorEn string
	Rating   int
	TextFa   string
	TextEn   string
}

type productSeed struct {
	Slug         string // PROVISIONAL (docs/API.md §8)
	NameFa       string
	NameEn       string
	Rating       float64
	ReviewCount  int
	BasePriceIrr int64 // price of the default storage tier, frontend toman × 10
	CompareGap   int64 // frontend oldPrice − price in IRR; 0 = no strike-through price
	SkuPrefix    string
	DefaultSku   string

	// Variants is built by init() from colors × tiers plus the
	// per-product stock overrides below.
	Colors      []colorSeed
	ColorCodes  []string // SKU code per Colors entry
	Tiers       []int    // storage tiers; Tiers[DefaultTier] carries BasePriceIrr
	DefaultTier int
	Stocks      []int         // base stock per Colors entry
	StockZero   [][2]int      // {colorIndex, tierIndex} pairs forced to stock 0
	Variants    []variantSeed // filled by init()

	Specs   []specSeed
	Reviews []reviewSeed
}

// Shipping methods: ids, names, descriptions and prices mirror the
// frontend checkout (fa/en). Express = 49,000 toman = 490,000 IRR;
// post is free.
var shippingMethods = []struct {
	ID            string
	NameFa        string
	NameEn        string
	DescriptionFa string
	DescriptionEn string
	PriceIrr      int64
}{
	{
		ID:            "express",
		NameFa:        "پیک اکسپرس تهران",
		NameEn:        "Tehran express courier",
		DescriptionFa: "تحویل در همان روز — فقط تهران",
		DescriptionEn: "Same-day delivery — Tehran only",
		PriceIrr:      490_000,
	},
	{
		ID:            "post",
		NameFa:        "پست پیشتاز",
		NameEn:        "Express post",
		DescriptionFa: "تحویل ۲ تا ۴ روز کاری — سراسر ایران",
		DescriptionEn: "Delivery in 2–4 business days — nationwide",
		PriceIrr:      0,
	},
}

// simSpec is shared by every product (frontend spec row for eSIM).
func simSpec() specSeed {
	return specSeed{
		LabelFa: "سیم‌کارت", LabelEn: "SIM",
		ValueFa: "سازگار با eSIM ایرانسل", ValueEn: "Compatible with Irancell eSIM",
	}
}

func skuTier(gb int) string {
	if gb >= 1000 {
		return "1TB"
	}
	return itoa(gb)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [8]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}

// expandVariants builds the color × tier matrix with SKU strings and
// deterministic stocks.
func (p *productSeed) expandVariants() {
	p.Variants = nil
	zero := map[[2]int]bool{}
	for _, z := range p.StockZero {
		zero[[2]int{z[0], z[1]}] = true
	}
	for ci, c := range p.Colors {
		for ti, gb := range p.Tiers {
			stock := p.Stocks[ci%len(p.Stocks)]
			if defaultStock, ok := defaultStocks[p.SkuPrefix]; ok && ci == 0 && ti == p.DefaultTier {
				stock = defaultStock
			}
			if zero[[2]int{ci, ti}] {
				stock = 0
			}
			p.Variants = append(p.Variants, variantSeed{
				Color:      c,
				StorageGb:  gb,
				TierOffset: ti - p.DefaultTier,
				StockQty:   stock,
				Sku:        p.SkuPrefix + "-" + p.ColorCodes[ci%len(p.ColorCodes)] + "-" + skuTier(gb),
			})
		}
	}
}

// defaultStocks pins the default variant's stock per product; the MVP
// product's default variant uses 7 to match the docs/API.md §4 example.
var defaultStocks = map[string]int{"IP15PM": 7}

var products = []productSeed{
	{
		// The MVP product: full 4-color × 4-storage variant matrix from
		// its detail page. Rating 4.8 mirrors the detail page text
		// (decimal — the §4.8 precision decision stays OPEN).
		Slug:         "iphone-15-pro-max",
		NameFa:       "آیفون ۱۵ پرو مکس",
		NameEn:       "iPhone 15 Pro Max",
		Rating:       4.8,
		ReviewCount:  246,
		BasePriceIrr: 489_900_000,
		CompareGap:   35_100_000,
		SkuPrefix:    "IP15PM",
		DefaultSku:   "IP15PM-NAT-256",

		Colors: []colorSeed{
			{Key: "natural", NameFa: "تیتانیوم طبیعی", NameEn: "Natural Titanium"},
			{Key: "blue", NameFa: "تیتانیوم آبی", NameEn: "Blue Titanium"},
			{Key: "black", NameFa: "تیتانیوم مشکی", NameEn: "Black Titanium"},
			{Key: "white", NameFa: "تیتانیوم سفید", NameEn: "White Titanium"},
		},
		ColorCodes:  []string{"NAT", "BLU", "BLK", "WHT"},
		Tiers:       []int{128, 256, 512, 1000},
		DefaultTier: 1, // 256 GB carries the catalog price
		Stocks:      []int{9, 8, 6, 5},
		StockZero:   [][2]int{{3, 3}}, // white 1 TB: zero-stock variant

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A17 Pro", ValueEn: "Apple A17 Pro"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۶٫۷ اینچ Super Retina XDR · ۱۲۰ هرتز", ValueEn: "6.7-inch Super Retina XDR · 120Hz"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۴۸ مگاپیکسل + ۱۲ اولتراواید + ۱۲ تله‌فوتو", ValueEn: "48MP + 12 ultrawide + 12 telephoto"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۴٬۴۴۱ میلی‌آمپرساعت · شارژ سریع ۲۷ وات", ValueEn: "4,441 mAh · 27W fast charging"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "تیتانیوم با محافظ Ceramic Shield", ValueEn: "Titanium with Ceramic Shield"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "سارا محمدی", AuthorEn: "Sara Mohammadi", Rating: 5,
				TextFa: "کیفیت دوربین فوق‌العاده است و باتری به‌راحتی یک روز کامل دوام می‌آورد. ارسال ایرانسل هم سریع بود.",
				TextEn: "The camera quality is outstanding and the battery easily lasts a full day. Irancell delivery was fast too."},
			{AuthorFa: "امیر رضایی", AuthorEn: "Amir Rezaei", Rating: 5,
				TextFa: "نسبت به نسل قبل خیلی سبک‌تر شده. صفحه‌نمایش در نور آفتاب هم عالی دیده می‌شود.",
				TextEn: "It is much lighter than the previous generation. The display looks great even in direct sunlight."},
			{AuthorFa: "نگار کریمی", AuthorEn: "Negar Karimi", Rating: 4,
				TextFa: "دستگاہ عالیه، فقط قیمتش کمی بالاست؛ البته تخفیف اقساطی ایرانسل خیلی کمک کرد.",
				TextEn: "Great device, though a bit pricey — the Irancell installment discount helped a lot."},
		},
	},
	{
		Slug:         "iphone-15-pro",
		NameFa:       "آیفون ۱۵ پرو",
		NameEn:       "iPhone 15 Pro",
		Rating:       5,
		ReviewCount:  189,
		BasePriceIrr: 435_000_000,
		CompareGap:   0,
		SkuPrefix:    "IP15P",
		DefaultSku:   "IP15P-BLU-256",

		Colors:      []colorSeed{{Key: "blue", NameFa: "تیتانیوم آبی", NameEn: "Blue Titanium"}},
		ColorCodes:  []string{"BLU"},
		Tiers:       []int{256},
		DefaultTier: 0,
		Stocks:      []int{6},

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A17 Pro", ValueEn: "Apple A17 Pro"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۶٫۱ اینچ Super Retina XDR · ۱۲۰ هرتز", ValueEn: "6.1-inch Super Retina XDR · 120Hz"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۴۸ مگاپیکسل + ۱۲ اولتراواید + ۱۲ تله‌فوتو", ValueEn: "48MP + 12 ultrawide + 12 telephoto"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۳٬۲۷۴ میلی‌آمپرساعت · شارژ سریع ۲۷ وات", ValueEn: "3,274 mAh · 27W fast charging"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "تیتانیوم با محافظ Ceramic Shield", ValueEn: "Titanium with Ceramic Shield"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "مهدی احمدی", AuthorEn: "Mehdi Ahmadi", Rating: 5,
				TextFa: "نسبت قیمت به عملکرد عالی است؛ برای عکاسی روزانه بهترین انتخاب بود.",
				TextEn: "Excellent price-to-performance; it was the best pick for everyday photography."},
			{AuthorFa: "الهام کریمی", AuthorEn: "Elham Karimi", Rating: 4,
				TextFa: "دستگاه کم‌نقص است، فقط دوست داشتم باتری بزرگ‌تری داشت.",
				TextEn: "Nearly flawless device — I only wish it had a bigger battery."},
		},
	},
	{
		Slug:         "iphone-15",
		NameFa:       "آیفون ۱۵",
		NameEn:       "iPhone 15",
		Rating:       4,
		ReviewCount:  132,
		BasePriceIrr: 354_000_000,
		CompareGap:   0,
		SkuPrefix:    "IP15",
		DefaultSku:   "IP15-PNK-128",

		Colors:      []colorSeed{{Key: "pink", NameFa: "صورتی", NameEn: "Pink"}},
		ColorCodes:  []string{"PNK"},
		Tiers:       []int{128},
		DefaultTier: 0,
		Stocks:      []int{9},

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A16 Bionic", ValueEn: "Apple A16 Bionic"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۶٫۱ اینچ Super Retina XDR", ValueEn: "6.1-inch Super Retina XDR"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۴۸ مگاپیکسل + ۱۲ اولتراواید", ValueEn: "48MP + 12 ultrawide"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۳٬۳۴۹ میلی‌آمپرساعت", ValueEn: "3,349 mAh"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "آلومینیوم با محافظ Ceramic Shield", ValueEn: "Aluminum with Ceramic Shield"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "رضا موسوی", AuthorEn: "Reza Mousavi", Rating: 4,
				TextFa: "برای استفاده روزمره کاملاً کافی است و دوربینش از انتظارم بهتر بود.",
				TextEn: "Plenty for everyday use, and the camera was better than I expected."},
			{AuthorFa: "مریم صادقی", AuthorEn: "Maryam Sadeghi", Rating: 4,
				TextFa: "طراحی زیبا و وزن مناسب؛ خرید اقساطی ایرانسل هم تجربه خوبی بود.",
				TextEn: "Beautiful design and comfortable weight; the Irancell installment plan was a good experience too."},
		},
	},
	{
		Slug:         "iphone-14-plus",
		NameFa:       "آیفون ۱۴ پلاس",
		NameEn:       "iPhone 14 Plus",
		Rating:       4,
		ReviewCount:  98,
		BasePriceIrr: 299_000_000,
		CompareGap:   16_000_000,
		SkuPrefix:    "IP14PL",
		DefaultSku:   "IP14PL-BLU-128",

		Colors:      []colorSeed{{Key: "blue", NameFa: "آبی", NameEn: "Blue"}},
		ColorCodes:  []string{"BLU"},
		Tiers:       []int{128},
		DefaultTier: 0,
		Stocks:      []int{5},

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A15 Bionic", ValueEn: "Apple A15 Bionic"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۶٫۷ اینچ Super Retina XDR", ValueEn: "6.7-inch Super Retina XDR"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۱۲ مگاپیکسل + ۱۲ اولتراواید", ValueEn: "12MP + 12 ultrawide"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۴٬۳۲۵ میلی‌آمپرساعت", ValueEn: "4,325 mAh"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "آلومینیوم با محافظ Ceramic Shield", ValueEn: "Aluminum with Ceramic Shield"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "علی نادری", AuthorEn: "Ali Naderi", Rating: 4,
				TextFa: "باتری بزرگش واقعاً راحت است؛ یک و نیم روز با یک شارژ کار می‌کند.",
				TextEn: "The big battery is a real relief — it runs a day and a half on one charge."},
			{AuthorFa: "شیما رستمی", AuthorEn: "Shima Rostami", Rating: 5,
				TextFa: "نمایشگر بزرگ برای فیلم دیدن عالی است و وزنش هم مشکلی ایجاد نمی‌کند.",
				TextEn: "The large screen is great for films and the weight is not a problem at all."},
		},
	},
	{
		Slug:         "iphone-14",
		NameFa:       "آیفون ۱۴",
		NameEn:       "iPhone 14",
		Rating:       4,
		ReviewCount:  157,
		BasePriceIrr: 275_000_000,
		CompareGap:   0,
		SkuPrefix:    "IP14",
		DefaultSku:   "IP14-BLK-128",

		Colors:      []colorSeed{{Key: "black", NameFa: "مشکی", NameEn: "Black"}},
		ColorCodes:  []string{"BLK"},
		Tiers:       []int{128},
		DefaultTier: 0,
		Stocks:      []int{8},

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A15 Bionic", ValueEn: "Apple A15 Bionic"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۶٫۱ اینچ Super Retina XDR", ValueEn: "6.1-inch Super Retina XDR"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۱۲ مگاپیکسل + ۱۲ اولتراواید", ValueEn: "12MP + 12 ultrawide"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۳٬۲۷۹ میلی‌آمپرساعت", ValueEn: "3,279 mAh"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "آلومینیوم با محافظ Ceramic Shield", ValueEn: "Aluminum with Ceramic Shield"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "حسین یاوری", AuthorEn: "Hossein Yavari", Rating: 4,
				TextFa: "گزینه‌ای مقرون‌به‌صرفه با کیفیت ساخت همیشگی اپل.",
				TextEn: "A budget-friendly option with Apple's usual build quality."},
			{AuthorFa: "پریسا اکبری", AuthorEn: "Parisa Akbari", Rating: 4,
				TextFa: "از دوتا نسل قبل ارتقا دادم؛ تفاوت سرعت کاملاً محسوس است.",
				TextEn: "Upgraded from two generations back; the speed difference is very noticeable."},
		},
	},
	{
		// Fully unavailable on purpose: search must still list it
		// (docs/API.md §2) and the seed set must exercise stock 0.
		Slug:         "iphone-se-3",
		NameFa:       "آیفون SE نسل سوم",
		NameEn:       "iPhone SE (3rd generation)",
		Rating:       4,
		ReviewCount:  64,
		BasePriceIrr: 198_000_000,
		CompareGap:   0,
		SkuPrefix:    "IPSE3",
		DefaultSku:   "IPSE3-WHT-128",

		Colors:      []colorSeed{{Key: "white", NameFa: "سفید ستاره‌ای", NameEn: "Starlight"}},
		ColorCodes:  []string{"WHT"},
		Tiers:       []int{128},
		DefaultTier: 0,
		Stocks:      []int{0},

		Specs: []specSeed{
			{LabelFa: "تراشه", LabelEn: "Chip", ValueFa: "Apple A15 Bionic", ValueEn: "Apple A15 Bionic"},
			{LabelFa: "نمایشگر", LabelEn: "Display", ValueFa: "۴٫۷ اینچ Retina HD", ValueEn: "4.7-inch Retina HD"},
			{LabelFa: "دوربین اصلی", LabelEn: "Main camera", ValueFa: "۱۲ مگاپیکسل", ValueEn: "12MP"},
			{LabelFa: "باتری", LabelEn: "Battery", ValueFa: "۲٬۰۱۸ میلی‌آمپرساعت", ValueEn: "2,018 mAh"},
			{LabelFa: "بدنه", LabelEn: "Body", ValueFa: "آلومینیوم با محافظ شیشه‌ای", ValueEn: "Aluminum with glass cover"},
			simSpec(),
		},
		Reviews: []reviewSeed{
			{AuthorFa: "کاوه شریفی", AuthorEn: "Kaveh Sharifi", Rating: 4,
				TextFa: "برای کسانی که گوشی جمع‌وجور می‌خواهند هنوز بهترین گزینه است.",
				TextEn: "Still the best pick for anyone who wants a compact phone."},
		},
	},
}

func init() {
	for i := range products {
		products[i].expandVariants()
	}
}
