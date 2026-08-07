-- +goose Up
-- +goose StatementBegin
CREATE VIEW quote_details AS
SELECT 
    q.id,
    q.customer_name,
    q.customer_phone,
    q.time_start,
    q.time_end,
    q.cart_id,
    q.request_type,
    q.status,
    q.comments,
    q.event_kind_id,
    ek.name AS event_kind_name,
    q.created_at,
    q.updated_at,
    COALESCE(
        json_agg(
            json_build_object(
                'id', ci.product_id,
                'name', p.name,
                'slug', p.slug,
                'category', ctg.name,
                'description', p.description,
                'quantity', ci.quantity,
                'max_quantity', p.quantity,
                'source', ci.source,
                'available', p.available,
                'image_url', i.filename
            )
        ) FILTER (WHERE ci.product_id IS NOT NULL),
        '[]'::json
    ) AS cart_items
FROM quotes q
LEFT JOIN cart_items ci ON q.cart_id = ci.cart_id
LEFT JOIN products p ON ci.product_id = p.id
LEFT JOIN categories ctg ON p.category = ctg.id
LEFT JOIN images i ON p.main_img = i.id
LEFT JOIN event_kinds ek ON q.event_kind_id = ek.id
GROUP BY 
    q.id, q.customer_name, q.customer_phone, q.time_start, 
    q.time_end, q.cart_id, q.request_type, q.status, 
    q.comments, q.event_kind_id, q.created_at, q.updated_at, ek.name;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP VIEW IF EXISTS quote_details;
-- +goose StatementEnd
