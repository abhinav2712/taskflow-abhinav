INSERT INTO users (id, name, email, password) VALUES
    (
        '00000000-0000-0000-0000-000000000001',
        'Test User',
        'test@example.com',
        '$2a$12$sf9BmwjYxMLmmbQ6C58ANOXhTiQ/TVMfD6N8biooLh3fB5VjwUYge'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO projects (id, name, description, owner_id) VALUES
    (
        '00000000-0000-0000-0000-000000000010',
        'Demo Project',
        'Seeded project for reviewers',
        '00000000-0000-0000-0000-000000000001'
    )
ON CONFLICT (id) DO NOTHING;

INSERT INTO tasks (
    id,
    title,
    description,
    status,
    priority,
    project_id,
    assignee_id,
    creator_id
) VALUES
    (
        '00000000-0000-0000-0000-000000000101',
        'Plan initial delivery',
        'First seeded task in todo state',
        'todo',
        'high',
        '00000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001'
    ),
    (
        '00000000-0000-0000-0000-000000000102',
        'Build project shell',
        'Second seeded task in progress',
        'in_progress',
        'medium',
        '00000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001'
    ),
    (
        '00000000-0000-0000-0000-000000000103',
        'Review seeded data',
        'Third seeded task already done',
        'done',
        'low',
        '00000000-0000-0000-0000-000000000010',
        '00000000-0000-0000-0000-000000000001',
        '00000000-0000-0000-0000-000000000001'
    )
ON CONFLICT (id) DO NOTHING;
