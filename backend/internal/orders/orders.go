// Package orders owns the order write model: the approved status
// enum and the persistence of orders + order items. Read APIs are
// future work (docs/API.md §7); in the MVP only checkout and the demo
// seeds write orders.
package orders

import (
	"context"
	"fmt"

	"backend/internal/platform/database"
)

// Approved order statuses (docs/API.md §4). Real checkout only ever
// creates `paid`; every other status exists exclusively on clearly
// marked demo records (is_demo = 1).
const (
	StatusPending    = "pending"
	StatusPaid       = "paid"
	StatusProcessing = "processing"
	StatusShipped    = "shipped"
	StatusDelivered  = "delivered"
	StatusCancelled  = "cancelled"
	StatusFailed     = "failed"
)

// ValidStatuses lists the approved lifecycle values in order.
var ValidStatuses = []string{
	StatusPending, StatusPaid, StatusProcessing, StatusShipped,
	StatusDelivered, StatusCancelled, StatusFailed,
}

// IsValidStatus reports whether s is an approved status.
func IsValidStatus(s string) bool {
	for _, v := range ValidStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// InsertItem is one order line, snapshotted at purchase time (both
// locales + unit price), per docs/API.md §4/§4.6.
type InsertItem struct {
	VariantID     int64
	ProductSlug   string
	ProductNameFa string
	ProductNameEn string
	ColorNameFa   string
	ColorNameEn   string
	StorageGb     int
	UnitPriceIrr  int64
	Quantity      int
	LineTotalIrr  int64
}

// Insert is the order write model: everything the orders table needs.
// SessionID 0 means no owning session (demo orders). Empty PaidAt
// means not paid; PaymentTransactionID empty means no payment attempt
// was recorded.
type Insert struct {
	TrackingCode         string
	SessionID            int64
	Status               string
	RecipientFirstName   string
	RecipientLastName    string
	RecipientPhone       string
	RecipientAddress     string
	ShippingMethodID     string
	ShippingNameFa       string
	ShippingNameEn       string
	ShippingPriceIrr     int64
	SubtotalIrr          int64
	TotalIrr             int64
	PaymentTransactionID string
	IsDemo               bool
	CreatedAt            string
	PaidAt               string
	Items                []InsertItem
}

// InsertOrder persists an order and its items inside the caller's
// transaction (checkout's BEGIN IMMEDIATE or the seed transaction).
func InsertOrder(ctx context.Context, tx database.Querier, o Insert) (int64, error) {
	if !IsValidStatus(o.Status) {
		return 0, fmt.Errorf("orders: invalid status %q", o.Status)
	}

	var sessionID any
	if o.SessionID > 0 {
		sessionID = o.SessionID
	}
	var paidAt any
	if o.PaidAt != "" {
		paidAt = o.PaidAt
	}
	var txn any
	if o.PaymentTransactionID != "" {
		txn = o.PaymentTransactionID
	}

	res, err := tx.ExecContext(ctx, `
		INSERT INTO orders
			(tracking_code, session_id, status,
			 recipient_first_name, recipient_last_name, recipient_phone, recipient_address,
			 shipping_method_id, shipping_name_fa, shipping_name_en, shipping_price_irr,
			 subtotal_irr, total_irr, payment_method, payment_transaction_id,
			 is_demo, created_at, paid_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'mock', ?, ?, ?, ?)`,
		o.TrackingCode, sessionID, o.Status,
		o.RecipientFirstName, o.RecipientLastName, o.RecipientPhone, o.RecipientAddress,
		o.ShippingMethodID, o.ShippingNameFa, o.ShippingNameEn, o.ShippingPriceIrr,
		o.SubtotalIrr, o.TotalIrr, txn,
		boolToInt(o.IsDemo), o.CreatedAt, paidAt,
	)
	if err != nil {
		return 0, fmt.Errorf("orders: insert: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("orders: insert id: %w", err)
	}

	const iq = `INSERT INTO order_items
		(order_id, variant_id, product_slug, product_name_fa, product_name_en,
		 color_name_fa, color_name_en, storage_gb, unit_price_irr, quantity, line_total_irr)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	for _, it := range o.Items {
		if _, err := tx.ExecContext(ctx, iq,
			id, it.VariantID, it.ProductSlug, it.ProductNameFa, it.ProductNameEn,
			it.ColorNameFa, it.ColorNameEn, it.StorageGb, it.UnitPriceIrr, it.Quantity,
			it.LineTotalIrr,
		); err != nil {
			return 0, fmt.Errorf("orders: insert item %s: %w", it.ProductSlug, err)
		}
	}
	return id, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
