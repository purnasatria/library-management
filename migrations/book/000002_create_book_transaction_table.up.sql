CREATE TABLE book_transactions (
    id UUID PRIMARY KEY,
    book_id UUID NOT NULL,
    user_id UUID NOT NULL,
    transaction_type VARCHAR(10) NOT NULL CHECK (transaction_type IN ('borrow', 'return')),
    transaction_date TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (book_id) REFERENCES books(id) ON DELETE CASCADE
);

CREATE INDEX idx_book_transactions_book_id ON book_transactions(book_id);
CREATE INDEX idx_book_transactions_user_id ON book_transactions(user_id);
CREATE INDEX idx_book_transactions_transaction_date ON book_transactions(transaction_date);


