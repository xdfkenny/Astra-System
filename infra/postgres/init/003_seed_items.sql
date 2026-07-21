-- Seed products for local development
-- Store: Astra Miami Brickell
-- Categories reference IDs from 001_schema.sql seed

INSERT INTO items (store_id, category_id, name, description, price_cents, cost_cents, image_url, tax_category, is_active) VALUES
    -- Produce
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Organic Gala Apples', 'Sweet and crisp, 3-count bag', 499, 250, 'https://picsum.photos/seed/apples/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Ripe Bananas', 'Cavendish bananas, per bunch', 299, 140, 'https://picsum.photos/seed/bananas/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Vine-Ripened Tomatoes', 'Roma tomatoes, 4-pack', 349, 175, 'https://picsum.photos/seed/tomatoes/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Hass Avocados', 'Medium ripe avocados, 2-count', 399, 200, 'https://picsum.photos/seed/avocados/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Romaine Lettuce Hearts', 'Crisp romaine hearts, 3-pack', 449, 220, 'https://picsum.photos/seed/lettuce/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9b-7000-b780-f75344a03381', 'Baby Carrots', 'Ready-to-eat peeled baby carrots, 1 lb', 299, 150, 'https://picsum.photos/seed/carrots/400/400', 'exempt', true),

    -- Bakery
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-a6e4-19ab818861c6', 'Artisan Sourdough Loaf', 'Slow-fermented with wild yeast, 24 oz', 699, 350, 'https://picsum.photos/seed/sourdough/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-a6e4-19ab818861c6', 'Butter Croissant', 'Flaky French-style croissant, baked daily', 399, 180, 'https://picsum.photos/seed/croissant/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-a6e4-19ab818861c6', 'French Baguette', 'Crisp crust, soft interior, 14 oz', 399, 160, 'https://picsum.photos/seed/baguette/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-a6e4-19ab818861c6', 'Blueberry Muffin', 'Loaded with wild blueberries, 4 oz', 349, 140, 'https://picsum.photos/seed/muffin/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-a6e4-19ab818861c6', 'Cinnamon Roll', 'Swirled with brown sugar and cream cheese icing', 449, 200, 'https://picsum.photos/seed/cinnamon/400/400', 'standard', true),

    -- Dairy
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-b9e5-96442f9b7704', 'Whole Milk', 'Fresh whole milk, half gallon', 499, 280, 'https://picsum.photos/seed/milk/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-b9e5-96442f9b7704', 'Sharp Cheddar Cheese', 'Aged 12 months, 8 oz block', 599, 320, 'https://picsum.photos/seed/cheddar/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-b9e5-96442f9b7704', 'Greek Yogurt', 'Plain non-fat, 32 oz tub', 599, 300, 'https://picsum.photos/seed/yogurt/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-b9e5-96442f9b7704', 'Large Brown Eggs', 'Free-range, dozen', 599, 350, 'https://picsum.photos/seed/eggs/400/400', 'exempt', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-b9e5-96442f9b7704', 'Unsalted Butter', 'European-style, 8 oz', 549, 280, 'https://picsum.photos/seed/butter/400/400', 'exempt', true),

    -- Beverages
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-9242-c1eb84e85b10', 'Fresh Orange Juice', 'Cold-pressed, 16 oz bottle', 599, 300, 'https://picsum.photos/seed/oj/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-9242-c1eb84e85b10', 'Sparkling Water', 'Natural mineral water, 1L', 249, 100, 'https://picsum.photos/seed/sparkling/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-9242-c1eb84e85b10', 'Nitro Cold Brew', 'Smooth cold brew infused with nitrogen, 12 oz can', 499, 220, 'https://picsum.photos/seed/coldbrew/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-9242-c1eb84e85b10', 'Matcha Green Tea', 'Japanese ceremonial grade, 16 oz', 449, 200, 'https://picsum.photos/seed/matcha/400/400', 'standard', true),
    ('550e8400-e29b-41d4-a716-446655440000', '019f85bb-6d9c-7000-9242-c1eb84e85b10', 'Fresh Lemonade', 'Hand-squeezed with organic lemons, 16 oz', 399, 160, 'https://picsum.photos/seed/lemonade/400/400', 'standard', true);

-- Seed inventory for each item
INSERT INTO inventory (store_id, item_id, quantity_available, reorder_point, reorder_quantity)
SELECT i.store_id, i.item_id, 100, 10, 50
FROM items i
WHERE i.store_id = '550e8400-e29b-41d4-a716-446655440000'
  AND NOT EXISTS (SELECT 1 FROM inventory inv WHERE inv.item_id = i.item_id);
