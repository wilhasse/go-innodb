# Users Table Data
Extracted directly from InnoDB file: `users.ibd`

## Table Structure
```sql
CREATE TABLE `users` (
  `id` int NOT NULL AUTO_INCREMENT,
  `name` varchar(100) DEFAULT NULL,
  `email` varchar(100) DEFAULT NULL,
  `created_at` timestamp NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (`id`)
) ENGINE=InnoDB AUTO_INCREMENT=4 DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_general_ci;
```

## Table Data

| ID | Name    | Email                   | Created At          |
|----|---------|-------------------------|---------------------|
| 1  | Alice   | alice@example.com       | 2025-08-17 22:22:37 |
| 2  | Bob     | bob@example.com         | 2025-08-17 22:22:37 |
| 3  | Charlie | charlie@example.com     | 2025-08-17 22:22:37 |

## Page Information
- Total pages: 7
- Data is stored in page 4 (INDEX page)
- 3 records found

## Files Generated
- `users_data.txt` - Raw tab-delimited data
- `users_with_headers.txt` - Tab-delimited data with column headers
- `users_table_report.md` - This formatted report