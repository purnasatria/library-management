CREATE TABLE category_items (
    id UUID PRIMARY KEY,
    category_id UUID NOT NULL,
    item_id UUID NOT NULL,
    item_type VARCHAR(50) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE CASCADE
);

CREATE INDEX idx_category_items_category_id ON category_items(category_id);
CREATE INDEX idx_category_items_item_id ON category_items(item_id);
CREATE INDEX idx_category_items_item_type ON category_items(item_type);
