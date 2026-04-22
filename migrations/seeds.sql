TRUNCATE TABLE tasks RESTART IDENTITY CASCADE;

INSERT INTO tasks (title, description, status, created_at, updated_at) VALUES
('Обход пациентов', 'Провести утренний обход пациентов в палатах 1-5', 'new', NOW(), NOW()),
('Связаться с пациентом Ивановым', 'Позвонить пациенту Иванову И.И. по результатам анализов', 'in_progress', NOW() - INTERVAL '2 hours', NOW() - INTERVAL '1 hour'),
('Подготовить отчёт', 'Сформировать ежемесячный отчёт по отделению', 'done', NOW() - INTERVAL '1 day', NOW() - INTERVAL '30 minutes');

INSERT INTO tasks (title, description, status, recurrence, created_at, updated_at) VALUES
('Ежедневный обзвон пациентов', 'Обзвонить пациентов из списка наблюдения', 'new', '{"type": "daily", "day_interval": 2}', NOW(), NOW()),
('Формирование отчётности', 'Подготовить отчёт по расходу медикаментов', 'new', '{"type": "monthly", "month_days": [1, 15]}', NOW(), NOW()),
('Плановая инвентаризация', 'Провести инвентаризацию медицинского оборудования', 'new', '{"type": "specific_dates", "specific_dates": ["2026-05-01", "2026-08-01", "2026-11-01"]}', NOW(), NOW()),
('Проверка журнала процедур', 'Проверить и подписать журнал выполненных процедур', 'new', '{"type": "even_odd_days", "even_odd_type": "even"}', NOW(), NOW()),
('Обновление карт пациентов', 'Внести актуальные данные в карты пациентов', 'new', '{"type": "even_odd_days", "even_odd_type": "odd"}', NOW(), NOW());
