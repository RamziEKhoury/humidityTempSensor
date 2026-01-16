                        ┌──────────────────────────┐
                        │         devices          │
                        ├──────────────────────────┤
                        │ id (PK, VARCHAR(64))     │
                        │ location VARCHAR(255)    │
                        └─────────────┬────────────┘
                                      │ 1
                                      │
                                      │ many
                        ┌─────────────┴────────────┐
                        │        readings          │
                        ├──────────────────────────┤
                        │ id (PK, BIGINT)          │
                        │ device_id (FK→devices.id)│
                        │ param_id  (FK→params.id) │
                        │ value DOUBLE             │
                        │ device_timestamp DATETIME│
                        │ received_at DATETIME     │
                        │ entry_hash BINARY(32)    │
                        │ UNIQUE(entry_hash)       │
                        └───────┬─────────┬────────┘
                                │         │
                                │         │ many
                                │         │
                                │         │
                                │         ▼
                     many       │   ┌───────────────┐
                                │   │     params    │
                                └──►│ (type catalog)│
                                    ├───────────────┤
                                    │ id (PK, INT)  │
                                    │ name VARCHAR  │
                                    └───────────────┘
