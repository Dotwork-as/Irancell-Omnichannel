package seed

import (
	"context"
	"fmt"

	"backend/internal/platform/database"
)

// Run applies all seeds inside one transaction. Every insert is
// insert-if-absent by natural key (slug / sku / shipping id /
// (product_id, position)); existing rows are never updated, so
// repeated startups never duplicate records and never reset stock or
// prices changed at runtime. Demo orders are seeded only when
// seedDemoOrders is true AND no demo order exists yet.
func Run(ctx context.Context, db *database.DB, seedDemoOrders bool) error {
	return db.WithTx(ctx, func(tx database.Querier) error {
		if err := seedShipping(ctx, tx); err != nil {
			return err
		}
		if err := seedCatalog(ctx, tx); err != nil {
			return err
		}
		if seedDemoOrders {
			if err := seedDemoOrdersData(ctx, tx); err != nil {
				return err
			}
		}
		return nil
	})
}

func seedShipping(ctx context.Context, tx database.Querier) error {
	const q = `INSERT INTO shipping_methods
		(id, name_fa, name_en, description_fa, description_en, price_irr, active)
		VALUES (?, ?, ?, ?, ?, ?, 1)
		ON CONFLICT (id) DO NOTHING`
	for _, m := range shippingMethods {
		if _, err := tx.ExecContext(ctx, q,
			m.ID, m.NameFa, m.NameEn, m.DescriptionFa, m.DescriptionEn, m.PriceIrr,
		); err != nil {
			return fmt.Errorf("seed: shipping method %s: %w", m.ID, err)
		}
	}
	return nil
}

func seedCatalog(ctx context.Context, tx database.Querier) error {
	for _, p := range products {
		if err := seedProduct(ctx, tx, p); err != nil {
			return err
		}
	}
	return nil
}

func seedProduct(ctx context.Context, tx database.Querier, p productSeed) error {
	const q = `INSERT INTO products
		(slug, name_fa, name_en, default_variant_id, rating, review_count)
		VALUES (?, ?, ?, 0, ?, ?)
		ON CONFLICT (slug) DO NOTHING`
	res, err := tx.ExecContext(ctx, q, p.Slug, p.NameFa, p.NameEn, p.Rating, p.ReviewCount)
	if err != nil {
		return fmt.Errorf("seed: product %s: %w", p.Slug, err)
	}
	inserted, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("seed: product %s: rows affected: %w", p.Slug, err)
	}

	var productID int64
	if err := tx.QueryRowContext(ctx,
		"SELECT id FROM products WHERE slug = ?", p.Slug).Scan(&productID); err != nil {
		return fmt.Errorf("seed: product %s: lookup id: %w", p.Slug, err)
	}

	for _, v := range p.Variants {
		price := p.BasePriceIrr + int64(v.TierOffset)*storageTierStepIrr
		var compare any
		if p.CompareGap > 0 {
			compare = price + p.CompareGap
		}
		const vq = `INSERT INTO product_variants
			(product_id, sku, color_key, color_name_fa, color_name_en,
			 storage_gb, price_irr, compare_at_price_irr, stock_qty)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (sku) DO NOTHING`
		if _, err := tx.ExecContext(ctx, vq,
			productID, v.Sku, v.Color.Key, v.Color.NameFa, v.Color.NameEn,
			v.StorageGb, price, compare, v.StockQty,
		); err != nil {
			return fmt.Errorf("seed: variant %s: %w", v.Sku, err)
		}
	}

	// default_variant_id can only be resolved after the variants
	// exist; set it once, on the insert that created this product.
	if inserted == 1 {
		var defaultID int64
		if err := tx.QueryRowContext(ctx,
			"SELECT id FROM product_variants WHERE sku = ?", p.DefaultSku).Scan(&defaultID); err != nil {
			return fmt.Errorf("seed: product %s: default variant %s: %w", p.Slug, p.DefaultSku, err)
		}
		if _, err := tx.ExecContext(ctx,
			"UPDATE products SET default_variant_id = ? WHERE id = ? AND default_variant_id = 0",
			defaultID, productID); err != nil {
			return fmt.Errorf("seed: product %s: set default variant: %w", p.Slug, err)
		}
	}

	const sq = `INSERT INTO product_specs
		(product_id, position, label_fa, label_en, value_fa, value_en)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT (product_id, position) DO NOTHING`
	for i, s := range p.Specs {
		if _, err := tx.ExecContext(ctx, sq,
			productID, i, s.LabelFa, s.LabelEn, s.ValueFa, s.ValueEn,
		); err != nil {
			return fmt.Errorf("seed: product %s: spec %d: %w", p.Slug, i, err)
		}
	}

	const rq = `INSERT INTO product_reviews
		(product_id, position, author_fa, author_en, rating, text_fa, text_en)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (product_id, position) DO NOTHING`
	for i, r := range p.Reviews {
		if _, err := tx.ExecContext(ctx, rq,
			productID, i, r.AuthorFa, r.AuthorEn, r.Rating, r.TextFa, r.TextEn,
		); err != nil {
			return fmt.Errorf("seed: product %s: review %d: %w", p.Slug, i, err)
		}
	}
	return nil
}
