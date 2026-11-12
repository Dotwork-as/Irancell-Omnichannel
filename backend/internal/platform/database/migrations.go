package database

// Migrations is the schema history of the application.
// Each migration lands inside its own transaction; see
// migrate.go. Domain notes:
//
//   - payments live on orders (payment_method/payment_transaction_id)
//     by design — there is no separate payments table.
//   - products.default_variant_id is a plain integer, not a DDL
//     foreign key: products and product_variants reference each
//     other, and SQLite cannot express that cycle in CREATE TABLE.
//     Seeds resolve it immediately after inserting the variants.
//   - prices are INTEGER rial everywhere; compare_at must stay ≥
//     price; order totals must satisfy total = subtotal + shipping.
var Migrations = []Migration{
	{
		ID:   1,
		Name: "catalog_tables",
		SQL: `
CREATE TABLE products (
	id                 INTEGER PRIMARY KEY,
	slug               TEXT NOT NULL UNIQUE,
	name_fa            TEXT NOT NULL,
	name_en            TEXT NOT NULL,
	default_variant_id INTEGER NOT NULL,
	rating             REAL NOT NULL CHECK (rating BETWEEN 0 AND 5),
	review_count       INTEGER NOT NULL DEFAULT 0 CHECK (review_count >= 0),
	created_at         TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE product_variants (
	id                   INTEGER PRIMARY KEY,
	product_id           INTEGER NOT NULL REFERENCES products(id),
	sku                  TEXT NOT NULL UNIQUE,
	color_key            TEXT NOT NULL,
	color_name_fa        TEXT NOT NULL,
	color_name_en        TEXT NOT NULL,
	storage_gb           INTEGER NOT NULL CHECK (storage_gb > 0),
	price_irr            INTEGER NOT NULL CHECK (price_irr >= 0),
	compare_at_price_irr INTEGER CHECK (compare_at_price_irr IS NULL OR compare_at_price_irr >= price_irr),
	stock_qty            INTEGER NOT NULL DEFAULT 0 CHECK (stock_qty >= 0),
	UNIQUE (product_id, color_key, storage_gb)
);
CREATE INDEX idx_product_variants_product ON product_variants(product_id);

CREATE TABLE product_specs (
	id         INTEGER PRIMARY KEY,
	product_id INTEGER NOT NULL REFERENCES products(id),
	position   INTEGER NOT NULL CHECK (position >= 0),
	label_fa   TEXT NOT NULL,
	label_en   TEXT NOT NULL,
	value_fa   TEXT NOT NULL,
	value_en   TEXT NOT NULL,
	UNIQUE (product_id, position)
);

CREATE TABLE product_reviews (
	id         INTEGER PRIMARY KEY,
	product_id INTEGER NOT NULL REFERENCES products(id),
	position   INTEGER NOT NULL CHECK (position >= 0),
	author_fa  TEXT NOT NULL,
	author_en  TEXT NOT NULL,
	rating     INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
	text_fa    TEXT NOT NULL,
	text_en    TEXT NOT NULL,
	UNIQUE (product_id, position)
);
`,
	},
	{
		ID:   2,
		Name: "shipping_methods",
		SQL: `
CREATE TABLE shipping_methods (
	id             TEXT PRIMARY KEY,
	name_fa        TEXT NOT NULL,
	name_en        TEXT NOT NULL,
	description_fa TEXT NOT NULL,
	description_en TEXT NOT NULL,
	price_irr      INTEGER NOT NULL CHECK (price_irr >= 0),
	active         INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1))
);
`,
	},
	{
		ID:   3,
		Name: "sessions_and_cart",
		SQL: `
CREATE TABLE guest_sessions (
	id         INTEGER PRIMARY KEY,
	token_hash TEXT NOT NULL UNIQUE,
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
	expires_at TEXT NOT NULL
);

CREATE TABLE carts (
	id         INTEGER PRIMARY KEY,
	session_id INTEGER NOT NULL UNIQUE REFERENCES guest_sessions(id),
	created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
	updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now'))
);

CREATE TABLE cart_items (
	id         INTEGER PRIMARY KEY,
	cart_id    INTEGER NOT NULL REFERENCES carts(id) ON DELETE CASCADE,
	variant_id INTEGER NOT NULL REFERENCES product_variants(id),
	quantity   INTEGER NOT NULL CHECK (quantity >= 1),
	UNIQUE (cart_id, variant_id)
);
CREATE INDEX idx_cart_items_cart ON cart_items(cart_id);
`,
	},
	{
		ID:   4,
		Name: "orders_and_items",
		SQL: `
CREATE TABLE orders (
	id                     INTEGER PRIMARY KEY,
	tracking_code          TEXT NOT NULL UNIQUE,
	session_id             INTEGER REFERENCES guest_sessions(id),
	status                 TEXT NOT NULL CHECK (status IN
		('pending','paid','processing','shipped','delivered','cancelled','failed')),
	recipient_first_name   TEXT NOT NULL,
	recipient_last_name    TEXT NOT NULL,
	recipient_phone        TEXT NOT NULL,
	recipient_address      TEXT NOT NULL,
	shipping_method_id     TEXT NOT NULL REFERENCES shipping_methods(id),
	shipping_name_fa       TEXT NOT NULL,
	shipping_name_en       TEXT NOT NULL,
	shipping_price_irr     INTEGER NOT NULL CHECK (shipping_price_irr >= 0),
	subtotal_irr           INTEGER NOT NULL CHECK (subtotal_irr >= 0),
	total_irr              INTEGER NOT NULL CHECK (total_irr = subtotal_irr + shipping_price_irr),
	payment_method         TEXT NOT NULL DEFAULT 'mock',
	payment_transaction_id TEXT,
	is_demo                INTEGER NOT NULL DEFAULT 0 CHECK (is_demo IN (0, 1)),
	created_at             TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ','now')),
	paid_at                TEXT
);
CREATE INDEX idx_orders_session ON orders(session_id);
CREATE INDEX idx_orders_status ON orders(status);
CREATE INDEX idx_orders_demo ON orders(is_demo);

CREATE TABLE order_items (
	id              INTEGER PRIMARY KEY,
	order_id        INTEGER NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
	variant_id      INTEGER NOT NULL REFERENCES product_variants(id),
	product_slug    TEXT NOT NULL,
	product_name_fa TEXT NOT NULL,
	product_name_en TEXT NOT NULL,
	color_name_fa   TEXT NOT NULL,
	color_name_en   TEXT NOT NULL,
	storage_gb      INTEGER NOT NULL,
	unit_price_irr  INTEGER NOT NULL CHECK (unit_price_irr >= 0),
	quantity        INTEGER NOT NULL CHECK (quantity >= 1),
	line_total_irr  INTEGER NOT NULL CHECK (line_total_irr = unit_price_irr * quantity)
);
CREATE INDEX idx_order_items_order ON order_items(order_id);
`,
	},
}
