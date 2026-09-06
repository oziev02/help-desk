package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/oziev02/help-desk/internal/domain"
)

func scanUser(row pgx.Row) (domain.User, error) {
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt)
	return u, err
}

func (p *Postgres) loadRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT r.name FROM roles r
		JOIN user_roles ur ON ur.role_id = r.id
		WHERE ur.user_id = $1
		ORDER BY r.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.Role
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		roles = append(roles, domain.Role(name))
	}
	return roles, rows.Err()
}

func (p *Postgres) CreateUser(ctx context.Context, email, passwordHash, fullName string, roles []domain.Role) (domain.User, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return domain.User{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var u domain.User
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, full_name)
		VALUES ($1, $2, $3)
		RETURNING id, email, password_hash, full_name, created_at`,
		email, passwordHash, fullName,
	).Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.User{}, domain.ErrEmailTaken
		}
		return domain.User{}, err
	}

	for _, role := range roles {
		if _, err := tx.Exec(ctx, `
			INSERT INTO user_roles (user_id, role_id)
			SELECT $1, id FROM roles WHERE name = $2`, u.ID, role); err != nil {
			return domain.User{}, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, err
	}
	u.Roles = roles
	return u, nil
}

func (p *Postgres) GetUserByID(ctx context.Context, id string) (domain.User, error) {
	u, err := scanUser(p.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, created_at
		FROM users WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	u.Roles, err = p.loadRoles(ctx, u.ID)
	return u, err
}

func (p *Postgres) GetUserByEmail(ctx context.Context, email string) (domain.User, error) {
	u, err := scanUser(p.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, full_name, created_at
		FROM users WHERE email = $1`, email))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.User{}, err
	}
	u.Roles, err = p.loadRoles(ctx, u.ID)
	return u, err
}

func (p *Postgres) ListExecutors(ctx context.Context) ([]domain.User, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT DISTINCT u.id, u.email, u.password_hash, u.full_name, u.created_at
		FROM users u
		JOIN user_roles ur ON ur.user_id = u.id
		JOIN roles r ON r.id = ur.role_id
		WHERE r.name = 'executor'
		ORDER BY u.full_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.User
	for rows.Next() {
		var u domain.User
		if err := rows.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.CreatedAt); err != nil {
			return nil, err
		}
		u.Roles = []domain.Role{domain.RoleExecutor}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (p *Postgres) GrantRole(ctx context.Context, userID string, role domain.Role) error {
	var roleID int
	err := p.pool.QueryRow(ctx, `SELECT id FROM roles WHERE name = $1`, role).Scan(&roleID)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.ErrInvalidInput
	}
	if err != nil {
		return err
	}
	_, err = p.pool.Exec(ctx, `
		INSERT INTO user_roles (user_id, role_id)
		VALUES ($1, $2)
		ON CONFLICT DO NOTHING`, userID, roleID)
	return err
}

func (p *Postgres) RevokeRole(ctx context.Context, userID string, role domain.Role) error {
	_, err := p.pool.Exec(ctx, `
		DELETE FROM user_roles
		WHERE user_id = $1 AND role_id = (SELECT id FROM roles WHERE name = $2)`,
		userID, role)
	return err
}

func (p *Postgres) ListRoles(ctx context.Context, userID string) ([]domain.Role, error) {
	return p.loadRoles(ctx, userID)
}

func (p *Postgres) EnsureDemoUsers(ctx context.Context) error {
	demos := []struct {
		email    string
		password string
		name     string
		roles    []domain.Role
	}{
		{"admin@helpdesk.local", "admin123", "Admin User", []domain.Role{domain.RoleAdmin}},
		{"user@helpdesk.local", "user123", "Regular User", []domain.Role{domain.RoleUser}},
		{"dispatcher@helpdesk.local", "dispatcher123", "Dispatcher User", []domain.Role{domain.RoleDispatcher}},
		{"executor@helpdesk.local", "executor123", "Executor User", []domain.Role{domain.RoleExecutor}},
		{"manager@helpdesk.local", "manager123", "Manager User", []domain.Role{domain.RoleManager}},
	}

	for _, d := range demos {
		_, err := p.GetUserByEmail(ctx, d.email)
		if err == nil {
			continue
		}
		if !errors.Is(err, domain.ErrNotFound) {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(d.password), bcrypt.DefaultCost)
		if err != nil {
			return err
		}
		if _, err := p.CreateUser(ctx, d.email, string(hash), d.name, d.roles); err != nil {
			return err
		}
	}
	return nil
}
