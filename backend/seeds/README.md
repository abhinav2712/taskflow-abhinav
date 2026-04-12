# Backend Seeds

These SQL files are reviewer-friendly copies of the backend seed data.

They mirror the same fixed UUID test data already applied by the migration pair:
- `backend/db/migrations/000002_seed.up.sql`
- `backend/db/migrations/000002_seed.down.sql`

Use them only after the schema migrations have already been applied.

## Apply test data manually

```bash
psql "$DATABASE_URL" -f backend/seeds/test_data.sql
```

## Remove the test data manually

```bash
psql "$DATABASE_URL" -f backend/seeds/cleanup.sql
```

Seeded credentials:
- Email: `test@example.com`
- Password: `password123`
