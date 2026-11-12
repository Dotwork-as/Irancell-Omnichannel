package seed

import (
	"context"
	"fmt"

	"backend/internal/platform/database"
)

// Demo orders: one per approved status, is_demo = 1, session_id NULL,
// clearly fake recipients, fixed spread timestamps (deterministic
// seeds). Real checkout orders can only ever be `paid`;
// every other status lives exclusively on these records.
// Tracking codes use a DEMO- prefix so they are distinguishable from
// real IRC- codes at a glance, on top of the is_demo flag.
type demoItemDef struct {
	SKU string
	Qty int
}

type demoOrderDef struct {
	TrackingCode  string
	Status        string
	CreatedAt     string
	PaidAt        string // "" → NULL (never paid)
	TransactionID string // "" → NULL (no payment attempt recorded)
	ShippingID    string
	First         string
	Last          string
	Phone         string
	Address       string
	Items         []demoItemDef
}

var demoOrders = []demoOrderDef{
	{
		TrackingCode: "DEMO-100001", Status: "pending",
		CreatedAt:  "2026-11-02T10:15:00Z",
		ShippingID: "post",
		First:      "نرگس", Last: "آزمایشی یک", Phone: "09120000001",
		Address: "تهران، خیابان نمونه، پلاک ۱، واحد ۱",
		Items:   []demoItemDef{{SKU: "IP15PM-NAT-256", Qty: 1}},
	},
	{
		TrackingCode: "DEMO-100002", Status: "paid",
		CreatedAt: "2026-11-03T11:20:00Z", PaidAt: "2026-11-03T11:21:00Z",
		TransactionID: "DEMO-TXN-100002",
		ShippingID:    "express",
		First:         "سعید", Last: "آزمایشی دو", Phone: "09120000002",
		Address: "تهران، خیابان نمونه، پلاک ۲، واحد ۲",
		Items:   []demoItemDef{{SKU: "IP15P-BLU-256", Qty: 1}, {SKU: "IP15PM-BLU-512", Qty: 1}},
	},
	{
		TrackingCode: "DEMO-100003", Status: "processing",
		CreatedAt: "2026-11-04T09:05:00Z", PaidAt: "2026-11-04T09:06:00Z",
		TransactionID: "DEMO-TXN-100003",
		ShippingID:    "post",
		First:         "الهه", Last: "آزمایشی سه", Phone: "09120000003",
		Address: "تهران، خیابان نمونه، پلاک ۳، واحد ۳",
		Items:   []demoItemDef{{SKU: "IP15-PNK-128", Qty: 2}},
	},
	{
		TrackingCode: "DEMO-100004", Status: "shipped",
		CreatedAt: "2026-11-05T14:40:00Z", PaidAt: "2026-11-05T14:41:00Z",
		TransactionID: "DEMO-TXN-100004",
		ShippingID:    "express",
		First:         "بابک", Last: "آزمایشی چهار", Phone: "09120000004",
		Address: "تهران، خیابان نمونه، پلاک ۴، واحد ۴",
		Items:   []demoItemDef{{SKU: "IP14PL-BLU-128", Qty: 1}},
	},
	{
		TrackingCode: "DEMO-100005", Status: "delivered",
		CreatedAt: "2026-11-06T08:30:00Z", PaidAt: "2026-11-06T08:31:00Z",
		TransactionID: "DEMO-TXN-100005",
		ShippingID:    "post",
		First:         "لیلا", Last: "آزمایشی پنج", Phone: "09120000005",
		Address: "تهران، خیابان نمونه، پلاک ۵، واحد ۵",
		Items:   []demoItemDef{{SKU: "IP14-BLK-128", Qty: 1}, {SKU: "IP15PM-NAT-128", Qty: 1}},
	},
	{
		TrackingCode: "DEMO-100006", Status: "cancelled",
		CreatedAt: "2026-11-07T16:10:00Z", PaidAt: "2026-11-07T16:11:00Z",
		TransactionID: "DEMO-TXN-100006",
		ShippingID:    "post",
		First:         "جمشید", Last: "آزمایشی شش", Phone: "09120000006",
		Address: "تهران، خیابان نمونه، پلاک ۶، واحد ۶",
		Items:   []demoItemDef{{SKU: "IP15PM-BLK-256", Qty: 1}},
	},
	{
		TrackingCode: "DEMO-100007", Status: "failed",
		CreatedAt:     "2026-11-08T12:00:00Z",
		TransactionID: "DEMO-TXN-100007",
		ShippingID:    "express",
		First:         "ژاله", Last: "آزمایشی هفت", Phone: "09120000007",
		Address: "تهران، خیابان نمونه، پلاک ۷، واحد ۷",
		Items:   []demoItemDef{{SKU: "IPSE3-WHT-128", Qty: 1}},
	},
}

func seedDemoOrdersData(ctx context.Context, tx database.Querier) error {
	var existing int
	if err := tx.QueryRowContext(ctx,
		"SELECT COUNT(*) FROM orders WHERE is_demo = 1").Scan(&existing); err != nil {
		return fmt.Errorf("seed: demo orders: count existing: %w", err)
	}
	if existing > 0 {
		return nil // never duplicated, never reset
	}

	var orderID int64
	for _, d := range demoOrders {
		shipping := findShipping(d.ShippingID)

		subtotal := int64(0)
		type resolvedItem struct {
			variantID            int64
			slug, nameFa, nameEn string
			colorFa, colorEn     string
			storageGb            int
			unitPrice            int64
			qty                  int
		}
		items := make([]resolvedItem, 0, len(d.Items))
		for _, it := range d.Items {
			var r resolvedItem
			err := tx.QueryRowContext(ctx, `
				SELECT pv.id, p.slug, p.name_fa, p.name_en,
				       pv.color_name_fa, pv.color_name_en, pv.storage_gb, pv.price_irr
				FROM product_variants pv
				JOIN products p ON p.id = pv.product_id
				WHERE pv.sku = ?`, it.SKU).Scan(
				&r.variantID, &r.slug, &r.nameFa, &r.nameEn,
				&r.colorFa, &r.colorEn, &r.storageGb, &r.unitPrice)
			if err != nil {
				return fmt.Errorf("seed: demo %s: variant %s: %w", d.TrackingCode, it.SKU, err)
			}
			r.qty = it.Qty
			subtotal += r.unitPrice * int64(r.qty)
			items = append(items, r)
		}

		var paidAt, txn any
		if d.PaidAt != "" {
			paidAt = d.PaidAt
		}
		if d.TransactionID != "" {
			txn = d.TransactionID
		}

		const oq = `INSERT INTO orders
			(tracking_code, session_id, status,
			 recipient_first_name, recipient_last_name, recipient_phone, recipient_address,
			 shipping_method_id, shipping_name_fa, shipping_name_en, shipping_price_irr,
			 subtotal_irr, total_irr, payment_method, payment_transaction_id,
			 is_demo, created_at, paid_at)
			VALUES (?, NULL, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'mock', ?, 1, ?, ?)`
		res, err := tx.ExecContext(ctx, oq,
			d.TrackingCode, d.Status,
			d.First, d.Last, d.Phone, d.Address,
			d.ShippingID, shipping.NameFa, shipping.NameEn, shipping.PriceIrr,
			subtotal, subtotal+shipping.PriceIrr, txn,
			d.CreatedAt, paidAt,
		)
		if err != nil {
			return fmt.Errorf("seed: demo order %s: %w", d.TrackingCode, err)
		}
		if orderID, err = res.LastInsertId(); err != nil {
			return fmt.Errorf("seed: demo order %s: insert id: %w", d.TrackingCode, err)
		}

		const iq = `INSERT INTO order_items
			(order_id, variant_id, product_slug, product_name_fa, product_name_en,
			 color_name_fa, color_name_en, storage_gb, unit_price_irr, quantity, line_total_irr)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
		for _, it := range items {
			if _, err := tx.ExecContext(ctx, iq,
				orderID, it.variantID, it.slug, it.nameFa, it.nameEn,
				it.colorFa, it.colorEn, it.storageGb, it.unitPrice, it.qty,
				it.unitPrice*int64(it.qty),
			); err != nil {
				return fmt.Errorf("seed: demo order %s: item %s: %w", d.TrackingCode, it.slug, err)
			}
		}
	}
	return nil
}

func findShipping(id string) struct {
	ID            string
	NameFa        string
	NameEn        string
	DescriptionFa string
	DescriptionEn string
	PriceIrr      int64
} {
	for _, m := range shippingMethods {
		if m.ID == id {
			return m
		}
	}
	panic("seed: unknown shipping method " + id)
}
