package database

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/valeriaulyamaeva/personal-finance-app/models"
)

func CreateReport(conn *pgx.Conn, report *models.Report) error {
	query := `
		INSERT INTO reports (user_id, report_data) 
		VALUES ($1, $2) 
		RETURNING id`

	err := conn.QueryRow(context.Background(), query,
		report.UserID,
		report.ReportData).Scan(&report.ID)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении отчета: %v", err)
	}
	return nil
}

func GetReportByID(conn *pgx.Conn, reportID int) (*models.Report, error) {
	query := `
		SELECT id, user_id, report_data, generated_date 
		FROM reports 
		WHERE id = $1`

	report := &models.Report{}
	err := conn.QueryRow(context.Background(), query, reportID).Scan(
		&report.ID,
		&report.UserID,
		&report.ReportData,
		&report.GeneratedDate,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("отчет с ID %d не найден", reportID)
		}
		return nil, fmt.Errorf("ошибка при получении отчета: %v", err)
	}

	return report, nil
}

func UpdateReport(conn *pgx.Conn, report *models.Report) error {
	query := `
		UPDATE reports 
		SET report_data = $1 
		WHERE id = $2`

	_, err := conn.Exec(context.Background(), query,
		report.ReportData,
		report.ID)
	if err != nil {
		return fmt.Errorf("ошибка обновления отчета: %v", err)
	}
	return nil
}

func DeleteReport(conn *pgx.Conn, reportID int) error {
	query := `
		DELETE FROM reports 
		WHERE id = $1`

	result, err := conn.Exec(context.Background(), query, reportID)
	if err != nil {
		return fmt.Errorf("ошибка удаления отчета: %v", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("отчет с ID %d не найден", reportID)
	}
	return nil
}
