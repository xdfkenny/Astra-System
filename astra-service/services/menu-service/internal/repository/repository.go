package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/astra-systems/astra-service/services/menu-service/internal/model"
	"github.com/astra-systems/astra-service/services/menu-service/internal/outbox"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Repository provides PostgreSQL access to the menu catalog using prepared statements.
type Repository struct {
	db *sql.DB

	// Read statements
	getCategoriesStmt      *sql.Stmt
	getItemsStmt           *sql.Stmt
	getItemByIDStmt        *sql.Stmt
	searchItemsStmt        *sql.Stmt
	getModifierGroupsStmt  *sql.Stmt

	// Write statements
	insertCategoryStmt   *sql.Stmt
	updateCategoryStmt   *sql.Stmt
	deleteCategoryStmt   *sql.Stmt
	insertItemStmt       *sql.Stmt
	updateItemStmt       *sql.Stmt
	updateItemPriceStmt  *sql.Stmt
	deleteItemStmt       *sql.Stmt
	getItemPriceStmt     *sql.Stmt
}

// NewRepository prepares all SQL statements.
func NewRepository(db *sql.DB) (*Repository, error) {
	r := &Repository{db: db}
	stmts := []struct {
		dest **sql.Stmt
		q    string
	}{
		{&r.getCategoriesStmt, `
			SELECT category_id, parent_id, name, description, display_order, image_url, blurhash, is_active
			FROM categories
			WHERE store_id = $1 AND deleted_at IS NULL AND ($2::boolean IS NULL OR is_active = $2)
			ORDER BY display_order, name
		`},
		{&r.getItemsStmt, `
			SELECT item_id, store_id, category_id, name, description, price_cents, cost_cents, plu, barcode, sku,
			       image_url, blurhash, tax_category, is_weight_based, weight_unit, is_active, metadata
			FROM items
			WHERE store_id = $1 AND deleted_at IS NULL AND ($2::boolean IS NULL OR is_active = $2)
			ORDER BY name
		`},
		{&r.getItemByIDStmt, `
			SELECT item_id, store_id, category_id, name, description, price_cents, cost_cents, plu, barcode, sku,
			       image_url, blurhash, tax_category, is_weight_based, weight_unit, is_active, metadata
			FROM items
			WHERE item_id = $1 AND deleted_at IS NULL
		`},
		{&r.searchItemsStmt, `
			SELECT item_id, store_id, category_id, name, description, price_cents, cost_cents, plu, barcode, sku,
			       image_url, blurhash, tax_category, is_weight_based, weight_unit, is_active, metadata
			FROM items
			WHERE store_id = $1 AND deleted_at IS NULL
			  AND ($2::boolean IS NULL OR is_active = $2)
			  AND ($3::text IS NULL OR name ILIKE $3 OR description ILIKE $3)
			  AND ($4::uuid IS NULL OR category_id = $4)
			ORDER BY name
		`},
		{&r.getModifierGroupsStmt, `
			SELECT img.item_id,
			       mg.modifier_group_id, mg.store_id, mg.name, mg.description, mg.min_select, mg.max_select, mg.display_order, mg.is_active,
			       mo.modifier_option_id, mo.name, mo.price_delta_cents, mo.is_default, mo.display_order, mo.is_active
			FROM item_modifier_groups img
			JOIN modifier_groups mg ON img.modifier_group_id = mg.modifier_group_id
			LEFT JOIN modifier_options mo ON mg.modifier_group_id = mo.modifier_group_id AND mo.deleted_at IS NULL
			WHERE img.item_id = ANY($1::uuid[]) AND mg.deleted_at IS NULL AND mg.is_active = true
			ORDER BY mg.display_order, mo.display_order
		`},
		{&r.insertCategoryStmt, `
			INSERT INTO categories (category_id, store_id, parent_id, name, description, display_order, image_url, blurhash, is_active)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`},
		{&r.updateCategoryStmt, `
			UPDATE categories
			SET parent_id = $2, name = $3, description = $4, display_order = $5, image_url = $6, blurhash = $7, is_active = $8
			WHERE category_id = $1 AND deleted_at IS NULL
		`},
		{&r.deleteCategoryStmt, `
			UPDATE categories SET deleted_at = NOW() WHERE category_id = $1 AND deleted_at IS NULL
		`},
		{&r.insertItemStmt, `
			INSERT INTO items (item_id, store_id, category_id, name, description, price_cents, cost_cents, plu, barcode, sku,
			                   image_url, blurhash, tax_category, is_weight_based, weight_unit, is_active, metadata)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		`},
		{&r.updateItemStmt, `
			UPDATE items
			SET category_id = $2, name = $3, description = $4, price_cents = $5, cost_cents = $6, plu = $7, barcode = $8, sku = $9,
			    image_url = $10, blurhash = $11, tax_category = $12, is_weight_based = $13, weight_unit = $14, is_active = $15, metadata = $16
			WHERE item_id = $1 AND deleted_at IS NULL
		`},
		{&r.updateItemPriceStmt, `
			UPDATE items SET price_cents = $2 WHERE item_id = $1 AND deleted_at IS NULL RETURNING store_id
		`},
		{&r.deleteItemStmt, `
			UPDATE items SET deleted_at = NOW() WHERE item_id = $1 AND deleted_at IS NULL
		`},
		{&r.getItemPriceStmt, `
			SELECT price_cents, store_id FROM items WHERE item_id = $1 AND deleted_at IS NULL
		`},
	}

	for _, s := range stmts {
		stmt, err := db.Prepare(s.q)
		if err != nil {
			return nil, fmt.Errorf("repository: prepare statement: %w", err)
		}
		*s.dest = stmt
	}
	return r, nil
}

// Close finalizes all prepared statements.
func (r *Repository) Close() error {
	var errs []error
	stmts := []*sql.Stmt{
		r.getCategoriesStmt, r.getItemsStmt, r.getItemByIDStmt, r.searchItemsStmt, r.getModifierGroupsStmt,
		r.insertCategoryStmt, r.updateCategoryStmt, r.deleteCategoryStmt,
		r.insertItemStmt, r.updateItemStmt, r.updateItemPriceStmt, r.deleteItemStmt, r.getItemPriceStmt,
	}
	for _, s := range stmts {
		if s != nil {
			if err := s.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("repository: close statements: %v", errs)
	}
	return nil
}

// GetCategories returns categories for a store, optionally including inactive ones.
func (r *Repository) GetCategories(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Category, error) {
	var activeFlag any
	if !includeInactive {
		activeFlag = true
	} else {
		activeFlag = nil
	}
	rows, err := r.getCategoriesStmt.QueryContext(ctx, storeID, activeFlag)
	if err != nil {
		return nil, fmt.Errorf("repository: get categories: %w", err)
	}
	defer rows.Close()

	var out []model.Category
	for rows.Next() {
		var c model.Category
		if err := rows.Scan(
			&c.CategoryID, &c.ParentID, &c.Name, &c.Description,
			&c.DisplayOrder, &c.ImageURL, &c.Blurhash, &c.IsActive,
		); err != nil {
			return nil, fmt.Errorf("repository: scan category: %w", err)
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate categories: %w", err)
	}
	return out, nil
}

// GetItems returns items for a store, optionally including inactive ones.
func (r *Repository) GetItems(ctx context.Context, storeID uuid.UUID, includeInactive bool) ([]model.Item, error) {
	var activeFlag any
	if !includeInactive {
		activeFlag = true
	} else {
		activeFlag = nil
	}
	rows, err := r.getItemsStmt.QueryContext(ctx, storeID, activeFlag)
	if err != nil {
		return nil, fmt.Errorf("repository: get items: %w", err)
	}
	defer rows.Close()

	var out []model.Item
	for rows.Next() {
		var it model.Item
		if err := r.scanItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate items: %w", err)
	}
	return out, nil
}

// GetItemByID returns a single item by id.
func (r *Repository) GetItemByID(ctx context.Context, itemID uuid.UUID) (*model.Item, error) {
	var it model.Item
	err := r.scanItem(r.getItemByIDStmt.QueryRowContext(ctx, itemID), &it)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("repository: get item by id: %w", err)
	}
	return &it, nil
}

// SearchItems searches items within a store.
func (r *Repository) SearchItems(ctx context.Context, storeID uuid.UUID, query, categoryID string, includeInactive bool) ([]model.Item, error) {
	var activeFlag any
	if !includeInactive {
		activeFlag = true
	} else {
		activeFlag = nil
	}
	var pattern any
	if query != "" {
		pattern = "%" + strings.ReplaceAll(query, "%", "\\%") + "%"
	}
	var catID any
	if categoryID != "" {
		id, err := uuid.Parse(categoryID)
		if err != nil {
			return nil, fmt.Errorf("repository: invalid category_id: %w", err)
		}
		catID = id
	}

	rows, err := r.searchItemsStmt.QueryContext(ctx, storeID, activeFlag, pattern, catID)
	if err != nil {
		return nil, fmt.Errorf("repository: search items: %w", err)
	}
	defer rows.Close()

	var out []model.Item
	for rows.Next() {
		var it model.Item
		if err := r.scanItem(rows, &it); err != nil {
			return nil, err
		}
		out = append(out, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate search items: %w", err)
	}
	return out, nil
}

// GetModifierGroupsByItemIDs returns modifier groups keyed by item id.
func (r *Repository) GetModifierGroupsByItemIDs(ctx context.Context, itemIDs []uuid.UUID) (map[uuid.UUID][]model.ModifierGroup, error) {
	if len(itemIDs) == 0 {
		return map[uuid.UUID][]model.ModifierGroup{}, nil
	}
	ids := make([]string, len(itemIDs))
	for i, id := range itemIDs {
		ids[i] = id.String()
	}
	rows, err := r.getModifierGroupsStmt.QueryContext(ctx, pgtype.FlatArray[string](ids))
	if err != nil {
		return nil, fmt.Errorf("repository: get modifier groups: %w", err)
	}
	defer rows.Close()

	out := make(map[uuid.UUID][]model.ModifierGroup)
	for rows.Next() {
		var itemID uuid.UUID
		var g model.ModifierGroup
		var optID, optName sql.NullString
		var optPrice sql.NullInt32
		var optDefault, optActive sql.NullBool
		var optOrder sql.NullInt32
		if err := rows.Scan(
			&itemID,
			&g.ModifierGroupID, &g.StoreID, &g.Name, &g.Description, &g.MinSelect, &g.MaxSelect, &g.DisplayOrder, &g.IsActive,
			&optID, &optName, &optPrice, &optDefault, &optOrder, &optActive,
		); err != nil {
			return nil, fmt.Errorf("repository: scan modifier group: %w", err)
		}

		groups := out[itemID]
		var existing *model.ModifierGroup
		for i := range groups {
			if groups[i].ModifierGroupID == g.ModifierGroupID {
				existing = &groups[i]
				break
			}
		}
		if existing == nil {
			groups = append(groups, g)
			existing = &groups[len(groups)-1]
		}
		if optID.Valid {
			existing.Options = append(existing.Options, model.ModifierOption{
				ModifierOptionID: uuid.MustParse(optID.String),
				ModifierGroupID:  g.ModifierGroupID,
				Name:             optName.String,
				PriceDeltaCents:  int(optPrice.Int32),
				IsDefault:        optDefault.Bool,
				DisplayOrder:     int(optOrder.Int32),
				IsActive:         optActive.Bool,
			})
		}
		out[itemID] = groups
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("repository: iterate modifier groups: %w", err)
	}
	return out, nil
}

// CreateCategory inserts a new category and emits a MenuUpdated outbox event.
func (r *Repository) CreateCategory(ctx context.Context, c *model.Category) error {
	if c.CategoryID == uuid.Nil {
		c.CategoryID = uuid.New()
	}
	now := time.Now().UTC()
	c.CreatedAt = now
	c.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin create category tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.StmtContext(ctx, r.insertCategoryStmt).ExecContext(ctx,
		c.CategoryID, c.StoreID, c.ParentID, c.Name, c.Description, c.DisplayOrder, c.ImageURL, c.Blurhash, c.IsActive,
	); err != nil {
		return fmt.Errorf("repository: insert category: %w", err)
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "category", c.CategoryID, c.StoreID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit create category: %w", err)
	}
	return nil
}

// UpdateCategory updates a category and emits a MenuUpdated outbox event.
func (r *Repository) UpdateCategory(ctx context.Context, c *model.Category) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin update category tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.StmtContext(ctx, r.updateCategoryStmt).ExecContext(ctx,
		c.CategoryID, c.ParentID, c.Name, c.Description, c.DisplayOrder, c.ImageURL, c.Blurhash, c.IsActive,
	)
	if err != nil {
		return fmt.Errorf("repository: update category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "category", c.CategoryID, c.StoreID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit update category: %w", err)
	}
	return nil
}

// DeleteCategory soft-deletes a category and emits a MenuUpdated outbox event.
func (r *Repository) DeleteCategory(ctx context.Context, categoryID, storeID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin delete category tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.StmtContext(ctx, r.deleteCategoryStmt).ExecContext(ctx, categoryID)
	if err != nil {
		return fmt.Errorf("repository: delete category: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "category", categoryID, storeID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit delete category: %w", err)
	}
	return nil
}

// CreateItem inserts a new item and emits a MenuUpdated outbox event.
func (r *Repository) CreateItem(ctx context.Context, it *model.Item) error {
	if it.ItemID == uuid.Nil {
		it.ItemID = uuid.New()
	}
	if it.TaxCategory == "" {
		it.TaxCategory = "standard"
	}
	now := time.Now().UTC()
	it.CreatedAt = now
	it.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin create item tx: %w", err)
	}
	defer tx.Rollback()

	meta, err := json.Marshal(it.Metadata)
	if err != nil {
		return fmt.Errorf("repository: marshal item metadata: %w", err)
	}
	if len(meta) == 0 || string(meta) == "null" {
		meta = []byte("{}")
	}

	if _, err := tx.StmtContext(ctx, r.insertItemStmt).ExecContext(ctx,
		it.ItemID, it.StoreID, it.CategoryID, it.Name, it.Description, it.PriceCents, it.CostCents, it.PLU, it.Barcode, it.SKU,
		it.ImageURL, it.Blurhash, it.TaxCategory, it.IsWeightBased, it.WeightUnit, it.IsActive, meta,
	); err != nil {
		return fmt.Errorf("repository: insert item: %w", err)
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "item", it.ItemID, it.StoreID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit create item: %w", err)
	}
	return nil
}

// UpdateItem updates an item and emits a MenuUpdated outbox event.
func (r *Repository) UpdateItem(ctx context.Context, it *model.Item) error {
	if it.TaxCategory == "" {
		it.TaxCategory = "standard"
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin update item tx: %w", err)
	}
	defer tx.Rollback()

	meta, err := json.Marshal(it.Metadata)
	if err != nil {
		return fmt.Errorf("repository: marshal item metadata: %w", err)
	}
	if len(meta) == 0 || string(meta) == "null" {
		meta = []byte("{}")
	}

	res, err := tx.StmtContext(ctx, r.updateItemStmt).ExecContext(ctx,
		it.ItemID, it.CategoryID, it.Name, it.Description, it.PriceCents, it.CostCents, it.PLU, it.Barcode, it.SKU,
		it.ImageURL, it.Blurhash, it.TaxCategory, it.IsWeightBased, it.WeightUnit, it.IsActive, meta,
	)
	if err != nil {
		return fmt.Errorf("repository: update item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "item", it.ItemID, it.StoreID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit update item: %w", err)
	}
	return nil
}

// UpdateItemPrice updates an item's price and emits an ItemPriceChanged outbox event.
func (r *Repository) UpdateItemPrice(ctx context.Context, itemID uuid.UUID, newPriceCents int) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin update price tx: %w", err)
	}
	defer tx.Rollback()

	var previousPrice int
	var storeID uuid.UUID
	if err := tx.StmtContext(ctx, r.getItemPriceStmt).QueryRowContext(ctx, itemID).Scan(&previousPrice, &storeID); err != nil {
		if err == sql.ErrNoRows {
			return sql.ErrNoRows
		}
		return fmt.Errorf("repository: get item price: %w", err)
	}

	res, err := tx.StmtContext(ctx, r.updateItemPriceStmt).ExecContext(ctx, itemID, newPriceCents)
	if err != nil {
		return fmt.Errorf("repository: update item price: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}

	if err := outbox.InsertItemPriceChanged(ctx, tx, itemID, storeID, int64(previousPrice), int64(newPriceCents)); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit update price: %w", err)
	}
	return nil
}

// DeleteItem soft-deletes an item and emits a MenuUpdated outbox event.
func (r *Repository) DeleteItem(ctx context.Context, itemID, storeID uuid.UUID) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("repository: begin delete item tx: %w", err)
	}
	defer tx.Rollback()

	res, err := tx.StmtContext(ctx, r.deleteItemStmt).ExecContext(ctx, itemID)
	if err != nil {
		return fmt.Errorf("repository: delete item: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	if err := outbox.InsertMenuUpdated(ctx, tx, "item", itemID, storeID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("repository: commit delete item: %w", err)
	}
	return nil
}

func (r *Repository) scanItem(sc interface{ Scan(dest ...any) error }, it *model.Item) error {
	var costCents sql.NullInt32
	var plu, barcode, sku, imageURL, blurhash, weightUnit sql.NullString
	var metadata []byte
	err := sc.Scan(
		&it.ItemID, &it.StoreID, &it.CategoryID, &it.Name, &it.Description,
		&it.PriceCents, &costCents, &plu, &barcode, &sku,
		&imageURL, &blurhash, &it.TaxCategory, &it.IsWeightBased, &weightUnit, &it.IsActive, &metadata,
	)
	if err != nil {
		return err
	}
	if costCents.Valid {
		v := int(costCents.Int32)
		it.CostCents = &v
	}
	if plu.Valid {
		it.PLU = &plu.String
	}
	if barcode.Valid {
		it.Barcode = &barcode.String
	}
	if sku.Valid {
		it.SKU = &sku.String
	}
	if imageURL.Valid {
		it.ImageURL = &imageURL.String
	}
	if blurhash.Valid {
		it.Blurhash = &blurhash.String
	}
	if weightUnit.Valid {
		it.WeightUnit = &weightUnit.String
	}
	if len(metadata) > 0 {
		_ = json.Unmarshal(metadata, &it.Metadata)
	}
	return nil
}
