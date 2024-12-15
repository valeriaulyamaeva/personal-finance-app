package database

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

// CreateFamilyAccount создает новый семейный аккаунт
func CreateFamilyAccount(pool *pgxpool.Pool, nickname string, ownerUserID int) (int, error) {
	var familyID int
	query := `
		INSERT INTO family_accounts (nickname, owner_user_id) 
		VALUES ($1, $2) 
		RETURNING id`
	err := pool.QueryRow(context.Background(), query, nickname, ownerUserID).Scan(&familyID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания семейного аккаунта: %v", err)
	}

	// Добавляем создателя как взрослого члена семьи
	memberQuery := `
		INSERT INTO family_memberships (user_id, family_account_id, role) 
		VALUES ($1, $2, 'adult')`
	_, err = pool.Exec(context.Background(), memberQuery, ownerUserID, familyID)
	if err != nil {
		return 0, fmt.Errorf("ошибка добавления создателя как члена семьи: %v", err)
	}

	return familyID, nil
}

// JoinFamilyAccount добавляет пользователя в существующий семейный аккаунт
func JoinFamilyAccount(pool *pgxpool.Pool, userID int, nickname string, role string) error {
	var familyID int
	query := `SELECT id FROM family_accounts WHERE nickname = $1`
	err := pool.QueryRow(context.Background(), query, nickname).Scan(&familyID)
	if err != nil {
		return fmt.Errorf("семейный аккаунт с ником '%s' не найден: %v", nickname, err)
	}

	memberQuery := `
		INSERT INTO family_memberships (user_id, family_account_id, role) 
		VALUES ($1, $2, $3)`
	_, err = pool.Exec(context.Background(), memberQuery, userID, familyID, role)
	if err != nil {
		return fmt.Errorf("ошибка добавления пользователя в семейный аккаунт: %v", err)
	}

	return nil
}

// GetFamilyMembers получает всех участников семейного аккаунта
func GetFamilyMembers(pool *pgxpool.Pool, familyAccountID int) ([]models.FamilyMember, error) {
	query := `
		SELECT u.id, u.name, f.role 
		FROM users u 
		JOIN family_memberships f ON u.id = f.user_id 
		WHERE f.family_account_id = $1`

	rows, err := pool.Query(context.Background(), query, familyAccountID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения участников семейного аккаунта: %v", err)
	}
	defer rows.Close()

	var members []models.FamilyMember
	for rows.Next() {
		var member models.FamilyMember
		if err := rows.Scan(&member.ID, &member.Name, &member.Role); err != nil {
			return nil, fmt.Errorf("ошибка при сканировании данных участника семьи: %v", err)
		}
		members = append(members, member)
	}

	return members, nil
}

// GetFamilyAccountByUser получает ID семейного аккаунта по ID пользователя
func GetFamilyAccountByUser(pool *pgxpool.Pool, userID int) (int, error) {
	var familyAccountID int
	query := `
		SELECT family_account_id 
		FROM family_memberships 
		WHERE user_id = $1`
	err := pool.QueryRow(context.Background(), query, userID).Scan(&familyAccountID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения семейного аккаунта для пользователя с ID %d: %v", userID, err)
	}

	return familyAccountID, nil
}

// GetFamilyAccountOwnerID получает ID владельца семейного аккаунта по ID семейного аккаунта
func GetFamilyAccountOwnerID(pool *pgxpool.Pool, familyAccountID int) (int, error) {
	var ownerID int
	query := `
		SELECT owner_user_id
		FROM family_accounts
		WHERE id = $1`
	err := pool.QueryRow(context.Background(), query, familyAccountID).Scan(&ownerID)
	if err != nil {
		return 0, fmt.Errorf("ошибка получения ID владельца семейного аккаунта: %v", err)
	}
	return ownerID, nil
}
